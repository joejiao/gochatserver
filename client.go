package main

import (
    "crypto/tls"
	"flag"
	"log"
	"net"
	"strconv"
	"time"
    //"bufio"
    "io"
    "io/ioutil"
    "math/rand"
    "math"
    //"net/http"
    _ "net/http/pprof"
)

var (
	hostAndPort = flag.String("host", "127.0.0.1:9999", "hostAndPort for server")
	maxconn     = flag.Int("maxconn", 2000, "max connector for server")
	qps         = flag.Int("qps", 10, "max connector for server")
    tt = make(chan bool)
)

func main() {
	flag.Parse()

    /*
    go func() {
        http.ListenAndServe("10.1.254.3:3339", nil)
    }()
    */

    ch := make(chan int, *maxconn)
    i := 1

    go sendTicker()

    for {
        ch <- i
        go connection(ch, i)
        time.Sleep(5 * time.Millisecond)
        i++
    }

    /*
	for i := 0; i < *maxconn; i++ {
		go connection(i)
        time.Sleep(10 * time.Millisecond)
	}

    for i := 0; i < *maxconn; i++ {
        fmt.Println(<-ch, "disconnected...")
    }
    */
}

func connection(ch chan int, i int) {
    conf := &tls.Config{
        InsecureSkipVerify: true,
    }

	conn, err := tls.Dial("tcp", *hostAndPort, conf)
    if err != nil {
        log.Fatal(err)
    }

	defer func() {
        log.Println("disconnect:", conn.LocalAddr().String())
        <-ch
	}()

    log.Println("connn to server:", conn.LocalAddr().String())

	//conn.SetLinger(0)
	//conn.SetKeepAlive(true)
	//conn.SetNoDelay(false)

    msg := strconv.Itoa(i)
    //roomName := strconv.Itoa(randInt(1,5))
    roomName := "1"

    if _, err := conn.Write([]byte("auth pw\n")); err != nil {
        log.Println("auth:", err)
        return
    }

    if _, err := conn.Write([]byte("uid " + strconv.Itoa(i) +"\n")); err != nil {
        log.Println("regsiter id:", err)
        return
    }

    joinRoom := "join room" + roomName + "\n"
    if _, err := conn.Write([]byte(joinRoom)); err != nil {
        log.Println("join room:", err)
        return
    }

    go sendMsg(conn, roomName, msg)
    readMsg(conn)
}

func readMsg(conn net.Conn) {
    defer conn.Close()
    io.Copy(ioutil.Discard, conn)

    /*
    reader := bufio.NewReader(conn)
    for {
        _, err := reader.ReadString('\n')
        if err != nil {
            log.Println(err)
            return
        }
        //fmt.Println(message)
    }
    */
}

func sendMsg(conn net.Conn, roomName string, msg string) {
    defer conn.Close()

    i := 1
    for _ = range tt {
        if i > 100 {
            return
        }

        m := "room: [" + roomName + "] " + msg + "-" + strconv.Itoa(i) + "\n"
        _, err := conn.Write([]byte(m))
        if err != nil {
            log.Println("write to server error:", err)
            return
        }
        i++
    }
}

func sendTicker() {
    sleepTime := math.Ceil(float64(1000 / *qps))
    ticker := time.NewTicker(time.Millisecond * time.Duration(sleepTime))

    for _ = range ticker.C {
        tt <- true
    }
}

func randInt(min int, max int) int {
    return min + rand.Intn(max-min)
}
