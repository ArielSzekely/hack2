CC=gcc
CFLAGS=-I.
LIBS=-lpthread
DEPS=%.h
SRC=ray-test.cc
TARGET=ray-test

all: $(TARGET)

%.o: %.cc $(DEPS)
	$(CC) -c -o $@ $< $(CFLAGS) $(LIBS)

$(TARGET): $(TARGET).o
	$(CC) $(CFLAGS) -o $(TARGET) $(TARGET).o $(LIBS)

run:
	./bench.sh

clean:
	rm -f *.o
	rm -f $(TARGET)
