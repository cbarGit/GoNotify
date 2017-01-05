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

/*watch descriptor struct*/
type wdesc struct {
	wd    uint32
	types uint32
}

/* inotify object struct */
type watchStruct struct {
	fd        int
	watchList map[string]*wdesc
	objects   map[uint32]string
}

/* inotify constants array */
var eventType = [7]uint32{unix.IN_CREATE, unix.IN_DELETE, unix.IN_MODIFY,
	unix.IN_MOVED_FROM, unix.IN_MOVED_TO, unix.IN_MOVE_SELF,
	unix.IN_DELETE_SELF}

/* IS_DIR constant */
const fileType uint32 = unix.IN_ISDIR

/* creates an inotify object and call readEvent on that object. */
func createWatch() (*watchStruct, error) {

	fd, err := unix.InotifyInit()
	if err != nil {
		fmt.Println(err)
	}

	inotifyObject := &watchStruct{
		fd:        fd,
		watchList: make(map[string]*wdesc),
		objects:   make(map[uint32]string),
	}

	go inotifyObject.readEvent()

	return inotifyObject, nil

}

/* add path to watchList */
func (inotifyObject *watchStruct) addWatch(d string) uint32 {
	wd, err := unix.InotifyAddWatch(inotifyObject.fd, d, unix.IN_ALL_EVENTS)
	if err != nil {
		fmt.Println(err)
	}
	inotifyObject.watchList[d] = &wdesc{wd: uint32(wd)}
	inotifyObject.objects[uint32(wd)] = d
	return uint32(wd)
}

/* remove path from watchList */
func (inotifyObject *watchStruct) rmWatch(d string) error {
	desc, ok := inotifyObject.watchList[d]
	if !ok {
		fmt.Println("error: cannot remove watch from watchList")
	}
	wd, err := unix.InotifyRmWatch(inotifyObject.fd, desc.wd)
	if wd == -1 {
		fmt.Println(err)
	}
	return nil
}

/* read events from buffer and eventually call detectEvent */
func (inotifyObject *watchStruct) readEvent() {
	var (
		p [unix.SizeofInotifyEvent * 4096]byte
		i uint32
	)

	for {
		length, err := unix.Read(inotifyObject.fd, p[:])
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
				path := inotifyObject.objects[wd]
				detectEvent(inotifyObject, wd, path, mask, eventType[:], i, p[:], lenN)

			}
			i += unix.SizeofInotifyEvent + lenN
		}
	}
}

/* detect the event, then print on terminal and call noty for libnotify notification */
func detectEvent(inotifyObject *watchStruct, wd uint32, path string, mask uint32, eventType []uint32, i uint32, p []byte, lenN uint32) {
	//fmt.Println(path)
	bytes := (*[unix.PathMax]byte)(unsafe.Pointer(
		&p[i+unix.SizeofInotifyEvent]))
	evName := strings.TrimRight(string(bytes[0:lenN]), "\000")
	if mask&eventType[0] != 0 {
		totName := path + "/" + evName
		if mask&fileType != 0 {
			fmt.Printf("New dir %v has been created in %v\n", evName, path)
			inotifyObject.addWatch(totName)
			noty(evName, "null", path, 0, 0)
		} else {
			fmt.Printf("New file %v has been created in %v\n", evName, path)
			inotifyObject.addWatch(totName)
			noty(evName, "null", path, 1, 0)
		}
	}
	if (mask&eventType[1] != 0) ||
		(mask&eventType[6] != 0) {
		if mask&fileType != 0 {
			fmt.Printf("Dir %v has been deleted in %v\n", evName, path)
			noty(evName, "null", path, 0, 1)
		} else {
			fmt.Printf("New file %v has been deleted in %v\n", evName, path)
			noty(evName, "null", path, 1, 1)
		}
	}
	if mask&eventType[6] != 0 {
		totName := evName + path
		if mask&fileType != 0 {
			fmt.Printf("Dir %v has been deleted\n", totName)
			noty(evName, "null", path, 0, 1)
		} else {
			fmt.Printf("File %v has been deleted\n", totName)
			noty(evName, "null", path, 1, 1)
		}
	}
	if mask&eventType[2] != 0 {
		if mask&fileType != 0 {
			fmt.Printf("New dir %v has been modified in %v\n", evName, path)
			noty(evName, "null", path, 0, 2)
		} else {
			fmt.Printf("New file %v has been modified in %v\n", evName, path)
			noty(evName, "null", path, 1, 2)
		}
	}
	if mask&eventType[3] != 0 {
		_, ok := inotifyObject.objects[wd]
		if ok {
			if mask&fileType != 0 {
				fmt.Printf("Dir %v has been moved from %v\n", evName, path)
				noty(evName, "null", path, 0, 3)
			} else {
				fmt.Printf("File %v has been moved from %v\n", evName, path)
				noty(evName, "null", path, 1, 3)
			}
		}
	}
	if mask&eventType[4] != 0 {
		_, ok := inotifyObject.objects[wd]
		if ok {
			if mask&fileType != 0 {
				fmt.Printf("Dir %v has been moved to %v\n", evName, path)
				noty(evName, "null", path, 0, 4)
			} else {
				fmt.Printf("File %v has been moved to %v\n", evName, path)
				noty(evName, "null", path, 1, 4)
			}
		}
	}
}

/* call libnotify for notification popup */
func noty(d string, d1 string, path string, types int, action int) {
	var text string
	var subj = [2]string{"Dir ", "File "}
	var act = [6]string{" created in ", " deleted in ", " modified in ", " moved from ", " moved to ", " renamed to "}

	notify.Init("GO-notify")

	if action < 5 {
		text = subj[types] + "\"" + d + "\"" + act[action] + "\"" + path + "\""
	} else {
		text = subj[types] + "\"" + d + "\"" + act[action] + "\"" + d1 + "\"" + " in folder " + "\"" + path + "\""
	}

	notific := notify.NotificationNew("GO-Notify", text, "")
	notify.NotificationSetTimeout(notific, 3000)
	if e := notify.NotificationShow(notific); e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e.Message())
		return
	}
	notify.NotificationClose(notific)
	notify.UnInit()
}

/* recursive walks through path, and add file/folder to watchList*/
func recList(dir string, wd *watchStruct) error {

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		absPath, _ := filepath.Abs(path)
		wd.addWatch(absPath)
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

/* main */
func main() {
	dir := os.Args[1]

	wd, err := createWatch()
	if err != nil {
		fmt.Println(err)
	}
	wd.addWatch(dir)
	err = recList(dir, wd)
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(wd.)
	wd.readEvent()

}
