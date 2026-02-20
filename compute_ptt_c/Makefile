CC      = gcc
CFLAGS  = -O2 -Wall -Wextra -pthread
LDFLAGS = -lduckdb
TARGET  = compute_ptt

$(TARGET): compute_ptt.c
	$(CC) $(CFLAGS) -o $(TARGET) compute_ptt.c $(LDFLAGS)

clean:
	rm -f $(TARGET)
