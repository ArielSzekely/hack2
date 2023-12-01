package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"os"
	"sync"
	"syscall"
)

const (
	SOCK_BACKLOG = 100
	SRV_PORT     = 8082
	SOCKET_MSG   = "hello socket!"
)

type Event struct {
	ID string `json:"id"`
}

type Result struct {
	Success bool   `json:"success"`
	Err     string `json:"err"`
}

func NewResult(name string, err error) *Result {
	var ok bool = true
	var errStr string
	if err != nil {
		ok = false
		errStr = err.Error()
	}
	return &Result{
		Success: ok,
		Err:     errStr,
	}
}

type Syscalls struct {
	sync.Mutex
	EventID string             `json:"id"`
	Results map[string]*Result `json:"results"`
}

func NewSyscalls(id string) *Syscalls {
	return &Syscalls{
		EventID: id,
		Results: make(map[string]*Result),
	}
}

func (sc *Syscalls) AddResult(name string, err error) {
	sc.Lock()
	defer sc.Unlock()

	sc.Results[name] = NewResult(name, err)
}

func (sc *Syscalls) socketClnt() {
	var fd int
	var sa *syscall.SockaddrInet4
	var n int
	var err error
	// Create an IPv4 socket.
	fd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err == nil {
		sa = &syscall.SockaddrInet4{
			Port: SRV_PORT,
			Addr: [4]byte{0, 0, 0, 0},
		}
		// Bind to the socket
		err = syscall.Connect(fd, sa)
		sc.AddResult("connect", err)
		if err == nil {
			n, err = syscall.Write(fd, []byte(SOCKET_MSG))
			sc.AddResult("socket.write", err)
			if n != len(SOCKET_MSG) {
				sc.AddResult("socket.write", fmt.Errorf("Wrong socket write size: %v != %v", n, len(SOCKET_MSG)))
			}
			err = syscall.Close(fd)
			sc.AddResult("socket_clnt.close", err)
		}
	}
}

func (sc *Syscalls) socketSrv() {
	var fd int
	var sa *syscall.SockaddrInet4
	var sa2 syscall.Sockaddr
	var nfd int
	var n int
	var b []byte = make([]byte, len(SOCKET_MSG))
	var err error
	// Create an IPv4 socket.
	fd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	sc.AddResult("socket", err)
	if err == nil {
		sa = &syscall.SockaddrInet4{
			Port: SRV_PORT,
			Addr: [4]byte{0, 0, 0, 0},
		}
		// Bind to the socket
		err = syscall.Bind(fd, sa)
		sc.AddResult("bind", err)
		if err == nil {
			err = syscall.Listen(fd, SOCK_BACKLOG)
			sc.AddResult("listen", err)
			if err == nil {
				// Start the client thread
				go sc.socketClnt()
				nfd, sa2, err = syscall.Accept(fd)
				sc.AddResult("accept", err)
				if err == nil {
					_ = sa2
					n, err = syscall.Read(nfd, b)
					sc.AddResult("socket.read", err)
					if n != len(SOCKET_MSG) {
						sc.AddResult("socket.read", fmt.Errorf("Wrong socket read size: %v != %v", n, len(SOCKET_MSG)))
					} else {
						if string(b) != SOCKET_MSG {
							sc.AddResult("socket.read", fmt.Errorf("Wrong socket message: %v != %v", string(b), SOCKET_MSG))
						}
					}
					err = syscall.Close(nfd)
					sc.AddResult("socket_srv.nfd.close", err)
				}
			}
		}
		err = syscall.Close(fd)
		sc.AddResult("socket_srv.close", err)
	}
}

func (sc *Syscalls) testSocketsLocal() {
	sc.socketSrv()
}

func (sc *Syscalls) Test() {
	// Test sockets
	sc.testSocketsLocal()
	// File
}

//func Chroot(path string) (err error)
//func Exec(argv0 string, argv []string, envv []string) (err error)
//func ForkExec(argv0 string, argv []string, attr *ProcAttr) (pid int, err error)
//func Getpgrp() (pid int)
//func Getpriority(which int, who int) (prio int, err error)
//func Getrlimit(resource int, rlim *Rlimit) (err error)
//func Getrusage(who int, rusage *Rusage) (err error)
//func Mlock(b []byte) (err error)
//func Mount(source string, target string, fstype string, flags uintptr, data string) (err error)
//func PivotRoot(newroot string, putold string) (err error)
//func PtraceAttach(pid int) (err error)
//func PtraceCont(pid int, signal int) (err error)
//func PtraceDetach(pid int) (err error)
//func PtraceGetEventMsg(pid int) (msg uint, err error)
//func PtraceGetRegs(pid int, regsout *PtraceRegs) (err error)
//func PtracePeekData(pid int, addr uintptr, out []byte) (count int, err error)
//func PtracePeekText(pid int, addr uintptr, out []byte) (count int, err error)
//func PtracePokeData(pid int, addr uintptr, data []byte) (count int, err error)
//func PtracePokeText(pid int, addr uintptr, data []byte) (count int, err error)
//func PtraceSetOptions(pid int, options int) (err error)
//func PtraceSetRegs(pid int, regs *PtraceRegs) (err error)
//func PtraceSingleStep(pid int) (err error)
//func PtraceSyscall(pid int, signal int) (err error)
//func Select(nfd int, r *FdSet, w *FdSet, e *FdSet, timeout *Timeval) (n int, err error)
//func Setrlimit(resource int, rlim *Rlimit) error
//func Setsid() (pid int, err error)

func HandleRequest(ctx context.Context, event *Event) (*string, error) {
	log.Printf("Handle request: %s", event.ID)
	defer log.Printf("Handle request done: %s", event.ID)
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}
	sc := NewSyscalls(event.ID)
	sc.Test()
	b, err := json.Marshal(sc)
	if err != nil {
		return nil, fmt.Errorf("Error marshal json: %v", err)
	}
	message := string(b)
	log.Printf(message)
	return &message, nil
}

func main() {
	if os.Getenv("LOCAL_DEV") == "" {
		lambda.Start(HandleRequest)
	} else {
		HandleRequest(context.TODO(), &Event{
			ID: "12345",
		})
	}
}
