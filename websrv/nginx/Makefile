CC=gcc
CFLAGS=-I. -g
LIBS=-lpthread -lfcgi
DEPS=%.h
SRC=srv.c
TARGET=srv

all: $(TARGET) run

%.o: %.c $(DEPS)
	$(CC) -c -o $@ $< $(CFLAGS) $(LIBS)

$(TARGET): $(TARGET).o
	$(CC) $(CFLAGS) -o $(TARGET) $(TARGET).o $(LIBS)

run:
	echo "abc"

clean:
	rm -f *.o
	rm -f $(TARGET)
