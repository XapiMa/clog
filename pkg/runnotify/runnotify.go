package runnotify

import (
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/xapima/conps/pkg/util"
)

type RunnotifyApi struct {
	Fs     *fsnotify.Watcher
	runCh  chan string
	killCh chan string
	errCh  chan error
}

func NewRunnotifyApi(runCh chan string, killCh chan string, errCh chan error) (*RunnotifyApi, error) {
	rapi := RunnotifyApi{Fs: nil, runCh: runCh, killCh: killCh, errCh: errCh}

	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, util.ErrorWrapFunc(err)
	}
	rapi.Fs = fs
	if err := rapi.addContaineRunMetrics(); err != nil {
		return nil, util.ErrorWrapFunc(err)
	}
	return &rapi, nil
}

func (rapi *RunnotifyApi) addContaineRunMetrics() error {
	if err := rapi.Fs.Add(runmetrics); err != nil {
		return util.ErrorWrapFunc(errors.Wrapf(err, "runmetrics %v is not add to watcher", runmetrics))
	}
	return nil
}

func (rapi *RunnotifyApi) Start() {
	defer close(rapi.runCh)
	defer close(rapi.killCh)
	defer close(rapi.errCh)
	lastRun := ""
	lastKill := ""
	for {
		select {
		case event := <-rapi.Fs.Events:
			switch {
			// case event.Op&fsnotify.Write == fsnotify.Write:
			// 	log.Printf("Write:  %s: %s", event.Op, event.Name)
			case event.Op&fsnotify.Create == fsnotify.Create:
				cid := filepath.Base(event.Name)
				if lastRun == cid {
					continue
				}
				rapi.runCh <- cid
				lastRun = cid

				// log.Printf("Create: %s: %s", event.Op, event.Name)
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				cid := filepath.Base(event.Name)
				if cid == lastKill {
					continue
				}
				rapi.killCh <- cid
				lastKill = cid

				// log.Printf("Remove: %s: %s", event.Op, event.Name)
				// case event.Op&fsnotify.Rename == fsnotify.Rename:
				// 	log.Printf("Rename: %s: %s", event.Op, event.Name)
				// case event.Op&fsnotify.Chmod == fsnotify.Chmod:
				// 	log.Printf("Chmod:  %s: %s", event.Op, event.Name)
			}
		case err := <-rapi.Fs.Errors:
			rapi.errCh <- err
		}
	}
}
