flags-scanner
=============

Are correct and "approved" compilation flags being used?

We scan the actual RPM packages to answer this question (and more).

Setup
-----

This project was developed, tested, and run on a Ubuntu system ;)

```
# On Ubuntu
sudo apt-get install librpm-dev libdw-dev libarchive-dev redis-server python-redis -y

# On Fedora
sudo dnf install elfutils-devel rpm-devel libarchive-devel python-redis redis -y
```

Use Go 1.5.1+ for compiling Go parts.

```
$ cat .bashrc
...

# Go Setup :-)
unset TMOUT
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$PATH
```

Usage
-----


``./scanner.py <target-folder>`` should do the trick.

Constraints
-----------

* Disk I/O is expensive (and I did not have any spare space earlier either), so
  we avoid all writes. As a positive side-effect, introducing parallelism
  becomes quite easy.

* We use a ``forking`` approach, since the underlying C libraries have known
  memory leak problems (librpm).

History
-------

This project got "delivered" 6 months late because the initial Python versions
ran into memory, and performance problems.

https://github.com/eliben/pyelftools/issues/26

Lately, I have seen ELF files of size greater than 2GiB. We really need some
speed to deal with such "monstrosities".

After more rewrites, we settled on the current design (forking approach with
the scanner written in Go). Ideally, we would like to use "libdw" and doing so
would cut the memory usage by half.

References
----------

* https://halfie.fedorapeople.org/flags-scanner/

* http://fedoraproject.org/wiki/Packaging:Guidelines#Compiler_flags

* http://fedoraproject.org/wiki/Hardened_Packages

* http://fedoraproject.org/wiki/Changes/FormatSecurity

* https://fedorahosted.org/fesco/ticket/1128 (-fstack-protector-strong)

* https://fedorahosted.org/fesco/ticket/1185 (-Werror=format-security)

* https://github.com/delano/redis-dump

* http://kylenubuntu.blogspot.se/2013/02/dbg-versus-dbgsym.html

* https://wiki.ubuntu.com/AptElfDebugSymbols

To-Do
-----

* Add support for scanning Debian (and Ubuntu) packages. They have crazy
  complex debugging setup.

* Integrate https://github.com/kholia/checksec into ``flags-scanner`` itself.
