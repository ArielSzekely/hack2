CC=gcc
CFLAGS=-I.
DEPS=%.h
SRC=spin.c
TARGET=spin

all: $(TARGET) run

%.o: %.c $(DEPS)
	$(CC) -c -o $@ $< $(CFLAGS)

$(TARGET): $(TARGET).o
	$(CC) -o $(TARGET) $(TARGET).o

run:
	./bench.sh

clean:
	rm -f *.o
	rm -f $(TARGET)
