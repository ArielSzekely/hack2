CC=gcc
CFLAGS=-I.
LIBS=-lpthread
DEPS=%.h
SRC=spin.c
TARGET=spin

all: $(TARGET) run

%.o: %.c $(DEPS)
	$(CC) -c -o $@ $< $(CFLAGS) $(LIBS)

$(TARGET): $(TARGET).o
	$(CC) $(CFLAGS) -o $(TARGET) $(TARGET).o $(LIBS)

run:
	./bench.sh

clean:
	rm -f *.o
	rm -f $(TARGET)
