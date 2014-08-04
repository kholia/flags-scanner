#!/usr/bin/env python2
# -*- coding: utf-8 -*-

import os
import shutil
import sys
import json
import subprocess
import traceback
import redis

# eu-readelf --debug-dump=info /usr/lib/debug/usr/bin/ls.debug  # on Fedora

DEBIAN = False
if DEBIAN:
    WORKER_BINARY = "deb-worker"
else:
    MAPPER_BINARY = "mapper"
    WORKER_BINARY = "worker"

# set of all "debuginfo" packages
debug_packages = set()

# set of all "normal" packages
packages = set()


def process_packages(packages, debug_packages):

    # flush everything!
    r = redis.Redis("localhost", db=13)
    r.flushall()

    jobs = {}
    if not DEBIAN:
        # build { "-debuginfo-" RPM(s) -> [package(s)] } => output of "mapper"
        p = subprocess.Popen(MAPPER_BINARY, stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE)
        data = {"packages": list(packages), "debug_packages": list(debug_packages)}
        data = json.dumps(data).encode('utf-8')
        out, _ = p.communicate(input=data)
        try:
            jobs = json.loads(out.decode("utf-8"))
        except ValueError:
            traceback.print_exc()
            return

    # process "jobs"
    for k, packages in jobs.items():
        p = subprocess.Popen(WORKER_BINARY, stdin=subprocess.PIPE,
                             stdout=subprocess.PIPE)
        data = {"debug_package": k, "packages": list(packages)}
        data = json.dumps(data).encode('utf-8')
        # print(data)
        out, _ = p.communicate(input=data)
        # print(out)
        try:
            jobs = json.loads(out.decode("utf-8"))
        except ValueError:
            traceback.print_exc()
            print("[-] was processing", data)

        r.set(k, out.decode("utf-8"))


def usage():
    print("Usage %s <folder containing packages>" % sys.argv[0])
    sys.exit(-1)


def main():
    if len(sys.argv) < 2:
        usage()

    target = sys.argv[1]
    if not os.path.isdir(target):
        usage()

    # is "mapper" available?
    if not DEBIAN:
        try:
            if not shutil.which(MAPPER_BINARY):
                sys.stderr.write("%s binary is missing, aborting!\n" %
                                 MAPPER_BINARY)
                sys.exit(-3)
        except AttributeError:
            # sys.stderr.write("shutil.which() is missing!\n")
            pass

    # make a set of all "debuginfo" and "normal" packages
    for root, dirs, files in os.walk(target):
        for name in files:
            realpath = os.path.join(root, name)
            if DEBIAN:
                if not os.path.isfile(realpath) or \
                        not (realpath.endswith(".deb") or realpath.endswith(".ddeb")):
                    continue
                if ".ddeb" in realpath:
                    debug_packages.add(realpath)
                else:
                    packages.add(realpath)
            else:
                if realpath.endswith(".noarch.rpm"):
                    continue
                if not os.path.isfile(realpath) or not realpath.endswith(".rpm"):
                    continue
                if "-debuginfo-" in realpath:
                    debug_packages.add(realpath)
                else:
                    packages.add(realpath)

    # obtain mapping from "debug_packages" to "packages"
    process_packages(packages, debug_packages)


if __name__ == "__main__":
    main()
