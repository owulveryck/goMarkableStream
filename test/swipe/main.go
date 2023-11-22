package main

import (
	"syscall"
	"time"
)

type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type SwipeDirection string

const (
	SwipeLeft  SwipeDirection = "Swipe Left"
	SwipeRight SwipeDirection = "Swipe Right"
)

func detectSwipe(chanEvent <-chan InputEvent, swipeC chan<- SwipeDirection) {
	const (
		evAbs        uint16 = 3
		codeXAxis    uint16 = 54
		minSwipeDist int32  = 250
		maxSwipeTime        = 800 * time.Millisecond
		maxStepDist  int32  = 25
	)

	var (
		startTime  time.Time
		startValue int32
		lastValue  int32
		isSwiping  bool
		swipeRight bool
	)

	for event := range chanEvent {
		if event.Type == evAbs && event.Code == codeXAxis {
			currentTime := timevalToTime(event.Time)
			if !isSwiping {
				startTime = currentTime
				startValue = event.Value
				lastValue = event.Value
				isSwiping = true
			} else {
				if abs(event.Value-lastValue) > maxStepDist {
					isSwiping = false
					continue
				}

				if event.Value > lastValue {
					swipeRight = true
				} else if event.Value < lastValue {
					swipeRight = false
				}

				if abs(event.Value-startValue) >= minSwipeDist && currentTime.Sub(startTime) <= maxSwipeTime {
					if swipeRight {
						swipeC <- SwipeRight
					} else {
						swipeC <- SwipeLeft
					}
					isSwiping = false
				}
				lastValue = event.Value
			}
		}
	}
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// timevalToTime converts syscall.Timeval to time.Time
func timevalToTime(tv syscall.Timeval) time.Time {
	return time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)
}
