#### Usage

Run the tool against `.debug` files. These `.debug` files can be
installed by the `debuginfo-install` command on Fedora and CentOS.

```
$ ./dwarf_producer /usr/lib/debug/usr/lib64/libgd.so.debug
/usr/lib/debug/usr/lib64/libgd.so.debug CU [18434] wbmp.c: GNU C ... -fstack-protector-strong
```

This tool has automatic support for dwz (https://fedoraproject.org/wiki/Features/DwarfCompressor).

Also see https://github.com/kholia/checksec/blob/master/libproducer.py project.

#### Credits

Huge thanks goes to mjw for all the code, and help.
