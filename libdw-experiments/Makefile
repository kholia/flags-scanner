demo:
	@gcc -Wall -g -O2 -o dwarf_producer dwarf_producer.c -ldw
	@gcc -Wall -g -O2 -o dwarfsrcfiles dwarfsrcfiles.c -ldw
	# @gcc -Wall -g -O2 -fsanitize=address -o dwarf_producer_memory dwarf_producer_memory.c -ldw -lrt
	# @gcc -Wall -g -O2 -o dwarf_producer_memory dwarf_producer_memory.c -ldw -lrt
	@gcc -g hello.c -o hello
	./dwarf_producer ./hello
	# ./dwarf_producer_memory ./hello

	@gcc -Wall -g -O2 -o dwarf_producer_memory-v2 dwarf_producer_memory-v2.c -ldw -lelf
	@./dwarf_producer_memory-v2 ./hello
