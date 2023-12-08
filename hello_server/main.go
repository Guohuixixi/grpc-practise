package main

import (
	"context"
	"fmt"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"hello_server/pb"
	"io"
	"net"
	"strings"
	"sync"
)

type server struct {
	pb.UnimplementedGreeterServer
	mu    sync.Mutex     //count的并发锁
	count map[string]int //记录每个name的请求次数
}

func magic(s string) string {
	s = strings.ReplaceAll(s, "吗", "")
	s = strings.ReplaceAll(s, "吧", "")
	s = strings.ReplaceAll(s, "你", "我")
	s = strings.ReplaceAll(s, "?", "!")
	s = strings.ReplaceAll(s, "?", "!")
	return s

}

func (s *server) BidiHello(stream pb.Greeter_BidiHelloServer) error {
	for {
		//接收流式请求
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		reply := magic(in.GetName())

		//返回流式响应
		if err := stream.Send(&pb.HelloResponse{Reply: reply}); err != nil {
			return err
		}
	}
}

func (s *server) LotsOfGreetings(stream pb.Greeter_LotsOfGreetingsServer) error {
	reply := "你好: "
	for {
		//接收客户端发来的流式数据
		res, err := stream.Recv()
		if err == io.EOF {
			//最终统一回复
			return stream.SendAndClose(&pb.HelloResponse{
				Reply: reply,
			})
		}
		if err != nil {
			return err
		}
		reply += res.GetName()
	}
}

func (s *server) LotsOfReplies(in *pb.HelloRequest, stream pb.Greeter_LotsOfRepliesServer) error {
	words := []string{
		"你好",
		"hello",
		"こんにちは",
		"안녕하세요",
	}
	for _, word := range words {
		data := &pb.HelloResponse{
			Reply: word + in.GetName(),
		}
		//使用send方法返回多个数据
		if err := stream.Send(data); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	//return &pb.HelloResponse{Reply: "Hello " + in.Name}, nil

	s.mu.Lock()
	defer s.mu.Unlock()
	s.count[in.Name]++ //记录请求次数
	//超过一次就返回i错误
	if s.count[in.Name] > 1 {
		st := status.New(codes.ResourceExhausted, "Requset limit exceeded.")
		ds, err := st.WithDetails(
			&errdetails.QuotaFailure{
				Violations: []*errdetails.QuotaFailure_Violation{{
					Subject:     fmt.Sprintf("name:%s", in.Name),
					Description: "限制每个name调用一次",
				}},
			})
		if err != nil {
			return nil, st.Err()
		}
		return nil, ds.Err()
	}
	//正常返回响应
	reply := "hello " + in.GetName()
	return &pb.HelloResponse{Reply: reply}, nil

}

func main() {
	//监听本地的8972端口
	lis, err := net.Listen("tcp", ":8972")
	if err != nil {
		fmt.Printf("failed to listen:%v", err)
		return
	}
	//使用服务器身份验证ssl/tls
	creds, _ := credentials.NewServerTLSFromFile("./server.crt", "./server.key")
	s := grpc.NewServer(grpc.Creds(creds))

	//s := grpc.NewServer()
	//pb.RegisterGreeterServer(s, &server{})
	pb.RegisterGreeterServer(s, &server{count: make(map[string]int)})
	//启动服务
	err = s.Serve(lis)
	if err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
}
