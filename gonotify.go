package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

type wdesc struct {
	wd    uint32
	types uint32
}

type watch_struct struct {
	fd         int
	watch_list map[string]*wdesc
}

var event_type = [7]uint32{unix.IN_CREATE, unix.IN_DELETE, unix.IN_MODIFY,
	unix.IN_MOVED_FROM, unix.IN_MOVED_TO, unix.IN_MOVE_SELF,
	unix.IN_DELETE_SELF}

var file_type uint32 = unix.IN_ISDIR

func create_watch() (*watch_struct, error) {

	fd, err := unix.InotifyInit()
	if err != nil {
		fmt.Println(err)
	}

	inotify_object := &watch_struct{
		fd:         fd,
		watch_list: make(map[string]*wdesc),
	}

	go inotify_object.readEvent()

	return inotify_object, nil

}

func (inotify_object *watch_struct) add_watch(d string) error {
	wd, err := unix.InotifyAddWatch(inotify_object.fd, d, event_type[0]|
		event_type[1]|event_type[2]|event_type[3]|event_type[4]|event_type[5]|
		event_type[6])
	if err != nil {
		fmt.Println(err)
	}
	inotify_object.watch_list[d] = &wdesc{wd: uint32(wd)}
	return nil
}

func (inotify_object *watch_struct) readEvent() {
	var (
		p [unix.SizeofInotifyEvent * 4096]byte
		i uint32
	)

	for {
		length, err := unix.Read(inotify_object.fd, p[:])
		if err != nil {
			fmt.Println(err)
		}

		i = 0

		for i < uint32(length-unix.SizeofInotifyEvent) {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&p[i]))

			mask := uint32(event.Mask)
			lenN := uint32(event.Len)

			if lenN != 0 {
				detectEvent(mask, event_type[:], i, p[:], lenN)
			}
			i += unix.SizeofInotifyEvent + lenN
		}
	}
}

func detectEvent(mask uint32, event_type []uint32, i uint32, p []byte, lenN uint32) {
	bytes := (*[unix.PathMax]byte)(unsafe.Pointer(
		&p[i+unix.SizeofInotifyEvent]))
	evName := strings.TrimRight(string(bytes[0:lenN]), "\000")
	if (mask & event_type[0]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been created\n", evName)
		} else {
			fmt.Printf("New file %v has been created\n", evName)
		}
	}
	if (mask & event_type[1]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been deleted\n", evName)
		} else {
			fmt.Printf("New file %v has been deleted\n", evName)
		}
	}
	if (mask & event_type[2]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been modified\n", evName)
		} else {
			fmt.Printf("New file %v has been modified\n", evName)
		}
	}
	if (mask & event_type[3]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been created\n", evName)
		} else {
			fmt.Printf("New file %v has been created\n", evName)
		}
	}
	if (mask & event_type[4]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been deleted\n", evName)
		} else {
			fmt.Printf("New file %v has been deleted\n", evName)
		}
	}
	if (mask & event_type[5]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("Dir %v has been moved\n", evName)
		} else {
			fmt.Printf("File %v has been moved\n", evName)
		}
	}
	if (mask & event_type[6]) != 0 {
		if mask&file_type != 0 {
			fmt.Printf("Dir %v has been deleted\n", evName)
		} else {
			fmt.Printf("File %v has been deleted\n", evName)
		}
	}
}

func recList(dir string, wd *watch_struct) error {

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			absPath, _ := filepath.Abs(path)
			wd.add_watch(absPath)
			fmt.Printf("DIR: %v.   (Just added to watch list)\n", absPath)
		} else {
			fmt.Printf("f |__ %v\n", filepath.Base(path))
		}
		return nil
	})
	fmt.Printf("\n")

	return nil
}

func main() {
	dir := os.Args[1]

	wd, err := create_watch()
	if err != nil {
		fmt.Println(err)
	}
	wd.add_watch(dir)
	err = recList(dir, wd)
	if err != nil {
		fmt.Println(err)
	}
	wd.readEvent()

}
