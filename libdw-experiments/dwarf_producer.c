// dwarf_producer.c - Get compiler flags with the dwarf in a elf file.
//
// gcc -Wall -g -O2 -lelf -ldw -o dwarfsrcfiles dwarfsrcfiles.c
//
// Copyright (C) 2011, Mark Wielaard <mjw@redhat.com>
// Copyright (C) 2017, Dhiru Kholia <dhiru@openwall.com>
//
// This file is free software.  You can redistribute it and/or modify
// it under the terms of the GNU General Public License (GPL); either
// version 2, or (at your option) any later version.

#include <argp.h>
#include <error.h>
#include <stdio.h>

#include <dwarf.h>
#include <elfutils/libdw.h>
#include <elfutils/libdwfl.h>

static int process_cu(Dwarf_Die * cu_die)
{
	Dwarf_Attribute attr;
	const char *producer;
	const char *name;

	if (dwarf_tag(cu_die) != DW_TAG_compile_unit) {
		// error(0, 0, "DIE isn't a compile unit");
		return 0;
	}

	if (dwarf_attr(cu_die, DW_AT_producer, &attr) == NULL)
		return -1;

	producer = dwarf_formstring(&attr);
	if (producer == NULL) {
		error(0, 0, "Couldn't get DW_AT_producer as string, %s",
		    dwarf_errmsg(-1));
		return -1;
	}
	name = dwarf_diename(cu_die);

	printf("CU %s: %s\n", (name ? : "<unknown>"), producer);

	return 0;
}

int main(int argc, char **argv)
{
	char *args[3];
	int res = 0;
	Dwfl *dwfl;
	Dwarf_Addr bias;

	if (argc != 2)
		error(-1, 0, "Usage %s <file>", argv[0]);

	// Pretend "dwarfsrcfiles -e <file>" was given, so we can use standard
	// dwfl argp parser to open the file for us and get our Dwfl. Useful
	// in case argument is an ET_REL file (like kernel modules). libdwfl
	// will fix up relocations for us.
	args[0] = argv[0];
	args[1] = "-e";
	args[2] = argv[1];

	argp_parse(dwfl_standard_argp(), 3, args, 0, NULL, &dwfl);

	Dwarf_Die *cu = NULL;
	while ((cu = dwfl_nextcu(dwfl, cu, &bias)) != NULL)
		res |= process_cu(cu);

	dwfl_end(dwfl);

	return res;
}
