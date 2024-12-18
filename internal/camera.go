package camera

import (
	"os"
)

func Capture() ([]byte, error) {
    return os.ReadFile("/dev/shm/mjpeg/cam.jpg")
}
