package client

import (
	"context"
	"image"
	"testing"

	"google.golang.org/grpc"
)

func BenchmarkGrabber_grab(b *testing.B) {
	grpcServer, ln := startStub()
	defer grpcServer.GracefulStop()
	//defer ln.Close()
	g := NewGrabber(&Configuration{}, &voidJpegDisplayer{})
	g.maxPictureGrabbed = 50
	conn, err := grpc.Dial(ln.Addr().String(), grpc.WithInsecure())
	if err != nil {
		b.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.imageHandler(ctx)
	for i := 0; i < b.N; i++ {
		err := g.grab(ctx, conn)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGrabber_grab(t *testing.T) {
	type fields struct {
		conf      *Configuration
		displayer Displayer
		imageC    chan *image.Gray
		rot       *rotation
		sleep     chan bool
	}
	type args struct {
		ctx  context.Context
		conn *grpc.ClientConn
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Grabber{
				conf:      tt.fields.conf,
				displayer: tt.fields.displayer,
				imageC:    tt.fields.imageC,
				rot:       tt.fields.rot,
				sleep:     tt.fields.sleep,
			}
			if err := g.grab(tt.args.ctx, tt.args.conn); (err != nil) != tt.wantErr {
				t.Errorf("Grabber.grab() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
