package main

import (
	"fmt"
)

func main() {
	//o := "/tmp/REb2b3c55e70f62e19a4f6d6fe0fc02d4b.mp3"
	o := "/tmp/RE8a3a4e3727f92aebb536a573385f5939_converted.wav"
	fmt.Printf("[i] Will open file: %s\n", o)
	t, err := Transcribe(o)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done")
	fmt.Println(t)
}
