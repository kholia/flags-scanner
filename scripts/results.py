#!/usr/bin/env python
# -*- coding: utf-8 -*-

import json
import redis
import traceback
import os


def main():
    r = redis.Redis("localhost", db=13)

    problematic_packages = set()
    for key in r.keys():
        try:
            files = json.loads(r[key])
            for f in files:
                stuff = files[f]
                if not stuff:
                    print "[!] empty producers for %s => %s" % (f, os.path.basename(key))
                    continue
                # print key, stuff
                for idx, val in enumerate(stuff):
                    if ("GNU " in val and not "GNU AS" in val) and ("strong" not in val and "-fstack-protector-all" not in val):
                        # trim "\x00" line terminator
                        v = val.strip("\x00")
                        print "[-] missing strong in %s => %s, %s" % (f, os.path.basename(key), v)
                        problematic_packages.add(key)
                    if "-O2" not in val and "GNU  " in val:
                        print "[*] missing -O2 in %s => %s, %s" % (f, os.path.basename(key), val)
                        problematic_packages.add(key)
                # print files.keys()

        except:
            print "[-] failed for", key
            traceback.print_exc()

    print problematic_packages


if __name__ == "__main__":
    main()
