package main

import notify "github.com/mqu/go-notify"

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
	objects    map[uint32]string
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
		objects:    make(map[uint32]string),
	}

	go inotify_object.readEvent()

	return inotify_object, nil

}

func (inotify_object *watch_struct) add_watch(d string) uint32 {
	wd, err := unix.InotifyAddWatch(inotify_object.fd, d, unix.IN_ALL_EVENTS)
	if err != nil {
		fmt.Println(err)
	}
	inotify_object.watch_list[d] = &wdesc{wd: uint32(wd)}
	inotify_object.objects[uint32(wd)] = d
	return uint32(wd)
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

			wd := uint32(event.Wd)
			mask := uint32(event.Mask)
			lenN := uint32(event.Len)

			if lenN != 0 {
				fmt.Println(event)
				detectEvent(inotify_object, wd, inotify_object.objects[wd], mask, event_type[:], i, p[:], lenN)
			}
			i += unix.SizeofInotifyEvent + lenN
		}
	}
}

func detectEvent(inotify_object *watch_struct, wd uint32, path string, mask uint32, event_type []uint32, i uint32, p []byte, lenN uint32) {
	//fmt.Println(path)
	bytes := (*[unix.PathMax]byte)(unsafe.Pointer(
		&p[i+unix.SizeofInotifyEvent]))
	evName := strings.TrimRight(string(bytes[0:lenN]), "\000")
	if mask&event_type[0] != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been created in %v\n", evName, path)
			noty(evName, path, 0, 0)
		} else {
			fmt.Printf("New file %v has been created in %v\n", evName, path)
			noty(evName, path, 1, 0)
		}
	}
	if (mask&event_type[1] != 0) ||
		(mask&event_type[6] != 0) {
		if mask&file_type != 0 {
			fmt.Printf("Dir %v has been deleted in %v\n", evName, path)
			noty(evName, path, 0, 1)
		} else {
			fmt.Printf("New file %v has been deleted in %v\n", evName, path)
			noty(evName, path, 1, 1)
		}
	}
	if mask&event_type[6] != 0 {
		totName := evName + path
		if mask&file_type != 0 {
			fmt.Printf("Dir %v has been deleted\n", totName)
			noty(evName, path, 0, 1)
		} else {
			fmt.Printf("File %v has been deleted\n", totName)
			noty(evName, path, 1, 1)
		}
	}
	if mask&event_type[2] != 0 {
		if mask&file_type != 0 {
			fmt.Printf("New dir %v has been modified in %v\n", evName, path)
			noty(evName, path, 0, 2)
		} else {
			fmt.Printf("New file %v has been modified in %v\n", evName, path)
			noty(evName, path, 1, 2)
		}
	}
	if mask&event_type[3] != 0 {
		_, ok := inotify_object.objects[wd]
		if ok {
			if mask&file_type != 0 {
				fmt.Printf("Dir %v has been moved from %v\n", evName, path)
				noty(evName, path, 0, 3)
			} else {
				fmt.Printf("File %v has been moved from %v\n", evName, path)
				noty(evName, path, 1, 3)
			}
		}
	}
	if mask&event_type[4] != 0 {
		_, ok := inotify_object.objects[wd]
		if ok {
			if mask&file_type != 0 {
				fmt.Printf("Dir %v has been moved to %v\n", evName, path)
				noty(evName, path, 0, 4)
			} else {
				fmt.Printf("File %v has been moved to %v\n", evName, path)
				noty(evName, path, 1, 4)
			}
		}
	}
}

func noty(d string, path string, types int, action int) {
	var text string
	var subj = [2]string{"Dir ", "File "}
	var act = [5]string{" created in ", " deleted in ", " modified in ", " moved from ", " moved to "}

	notify.Init("GO-notify")

	text = subj[types] + d + act[action] + path

	notific := notify.NotificationNew("GO-Notify", text, "")
	notify.NotificationSetTimeout(notific, 3000)
	if e := notify.NotificationShow(notific); e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e.Message())
		return
	}
	notify.NotificationClose(notific)
	notify.UnInit()
}

func recList(dir string, wd *watch_struct) error {

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		absPath, _ := filepath.Abs(path)
		wd.add_watch(absPath)
		if info.IsDir() {
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
	//fmt.Println(wd.)
	wd.readEvent()

}
