package camera

import (
	"io"
	"os"
)

func Capture() error {
    src := "/dev/shm/mjpeg/cam.jpg"
    dst := "../data/frame.jpg"
    source, err := os.Open(src)
    if err != nil {
        return err
    }
    defer source.Close()

    destination, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destination.Close()

    _, err = io.Copy(destination, source)   
    if err != nil {
        return err
    }

    return nil
}
