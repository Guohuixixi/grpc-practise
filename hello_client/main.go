package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"hello_client/pb"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultName = "xixi"
)

var (
	addr = flag.String("addr", "127.0.0.1:8972", "the address to connect")
	name = flag.String("name", defaultName, "Name to greet")
)

func main() {
	flag.Parse()
	creds, _ := credentials.NewClientTLSFromFile("./server.crt", "")
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(creds))
	//conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	runSayHello(c)
	//runLotsOfReplies(c)
	//runLotsOfGreetings(c)
	//runBidiHello(c)

}

func runBidiHello(c pb.GreeterClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	stream, err := c.BidiHello(ctx)
	if err != nil {
		log.Fatalf("c.BidiHello failed, err: %v", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			//接收服务端返回的响应
			in, err := stream.Recv()
			if err == io.EOF {
				//read done
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("c.BidiHello stream.Recv() failed, err: %v", err)
			}
			fmt.Printf("AI:%s\n", in.GetReply())
		}
	}()
	//从标准输入获取用户输入
	fmt.Println("请开始你的对话:")
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)
		if len(cmd) == 0 {
			continue
		}
		if strings.ToUpper(cmd) == "QUIT" {
			break
		}
		//将获取到的数据发送至服务端
		if err := stream.Send(&pb.HelloRequest{Name: cmd}); err != nil {
			log.Fatalf("c.BidiHello stream.Send(%v) failed, err: %v", cmd, err)
		}
	}
	stream.CloseSend()
	<-waitc
}

func runLotsOfGreetings(c pb.GreeterClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//客户端流式RPC
	stream, err := c.LotsOfGreetings(ctx)
	if err != nil {
		log.Fatalf("c.LotsOfGreetings failed, err: %v", err)
	}
	names := []string{"李国辉", "lgh", "世界"}
	for _, name := range names {
		//发送流式数据
		err := stream.Send(&pb.HelloRequest{Name: name})
		if err != nil {
			log.Fatalf("c.LotsOfGreetings failed, err: %v", err)
		}
	}
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("c.LotsOfGreetings failed, err: %v", err)
	}
	log.Printf("got reply: %v", res.GetReply())

}

func runSayHello(c pb.GreeterClient) {
	//执行RPC调用并打印收到的响应数据
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetReply())
}
func runLotsOfReplies(c pb.GreeterClient) {
	//server端流式RPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := c.LotsOfReplies(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("c.LotsOfReplies failed,err: %v", err)
	}
	for {
		//接收服务端返回的流式数据，当收到io.EOF或错误时退出
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("c.LotsOfReplies failed,err: %v", err)
		}
		log.Printf("got reply: %q\n", res.GetReply())
	}
}
