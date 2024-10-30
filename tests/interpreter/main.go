package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AirspaceStatus holds the status of the CTR and TMA areas
type AirspaceStatus struct {
	Areas      []Area `json:"areas"`
	NextUpdate string `json:"nextUpdate"`
}

// Area represents the status and next update time of an airspace component
type Area struct {
	Name       string `json:"name"`
	Status     bool   `json:"status"`
	NextUpdate string `json:"nextUpdate"`
}

type TimeSegment struct {
	Type  string
	Times []time.Time
}

// parseTranscript parses the provided transcript and extracts the airspace status
func parseAirspaceStates(transcript string) (AirspaceStatus, error) {
	transcript = strings.ToLower(transcript)

	// Regular expressions to capture information
	statusRegex := regexp.MustCompile(`(meiringen\s+ctr\s+and\s+(all\s+tma\s+sectors|tma)\s+are\s+active|not\s+active)`)
	//updateRegex := regexp.MustCompile(`updated\s+at\s+([\d\s:]+)`)
	nextUpdateRegex := regexp.MustCompile(`next\s+update\s+(at\s+[\d\s:]+|monday\s+[\d\s:]+)`)
	//operatingHoursRegex := regexp.MustCompile(`flight\s+operating\s+hours\s+are\s+from\s+([\d\s:]+)\s+till\s+([\d\s:]+)`)

	// Default areas
	areas := []Area{
		{"Meiringen CTR", false, ""},
		{"Meiringen TMA 1", false, ""},
		{"Meiringen TMA 2", false, ""},
		{"Meiringen TMA 3", false, ""},
		{"Meiringen TMA 4", false, ""},
		{"Meiringen TMA 5", false, ""},
		{"Meiringen TMA 6", false, ""},
	}

	// Set default next update time
	nextUpdate := "Unknown"

	// Extract the status
	statusMatch := statusRegex.FindStringSubmatch(transcript)
	if len(statusMatch) > 0 {
		status := strings.Contains(statusMatch[0], "active")
		for i := range areas {
			if status {
				areas[i].Status = true
			} else {
				areas[i].Status = false
			}
		}
	}

	// Extract the next update time
	nextUpdateMatch := nextUpdateRegex.FindStringSubmatch(transcript)
	if len(nextUpdateMatch) > 0 {
		nextUpdate = nextUpdateMatch[1]
		nextUpdate = strings.ReplaceAll(nextUpdate, "at ", "")
		nextUpdate = strings.TrimSpace(nextUpdate)
	}

	// Apply next update to all areas
	for i := range areas {
		areas[i].NextUpdate = nextUpdate
	}

	// Return the parsed data
	return AirspaceStatus{
		Areas:      areas,
		NextUpdate: nextUpdate,
	}, nil
}

func parseTimeToCurrentDate(timeString string) (time.Time, error) {
	parsedTime, err := time.Parse("15:04", timeString)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing time: %w", err)
	}

	now := time.Now()

	// Combine the parsed time with the current date in the local timezone
	finalTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		parsedTime.Hour(), parsedTime.Minute(), 0, 0,
		now.Location(),
	)

	return finalTime, nil
}

// Extract time segments
func ParseTimeSegments(transcript string) []TimeSegment {
	patternTimeSegments := `(\d{1,2}[: ]\d{2})`

	// Split all time segments by the "local time" substring
	// segment 1: Message update times
	// segment 2: Flight operating hours morning
	// segment 3: Flight operating hours evening
	// [[t1, t2], [t1, t2], [t1, t2]]
	var timeSegments [][]time.Time

	/*
		Check if this transcript is for the weekend (only one update time)
		`Meiringen ctr and tma are not active
		expect ctr and tma meiringen to be active again next monday through 7 30 local time.
		if you hear this message on monday after 07 30 local time, contact...`
	*/
	onlyOneUpdateTime := strings.Contains(transcript, "be active again next")

	splitLocalTime := strings.Split(transcript, "local time")
	for i, split := range splitLocalTime {
		if onlyOneUpdateTime && i == 1 {
			continue
		}

		trimmed := strings.TrimSpace(split)
		if regexp.MustCompile(`\d`).MatchString(trimmed) {
			re := regexp.MustCompile(patternTimeSegments)
			matches := re.FindAllStringSubmatch(trimmed, -1)

			// Parse each time segment within local time segment
			var segments []time.Time
			for _, match := range matches {
				times := regexp.MustCompile(`\d{1,2}[: ]\d{2}`).FindAllString(match[1], -1)
				for _, timeString := range times {
					replacedString := strings.Replace(timeString, " ", ":", 1)
					convertedTime, err := parseTimeToCurrentDate(replacedString)
					if err != nil {
						panic(err)
					}

					segments = append(segments, convertedTime)
				}
			}

			timeSegments = append(timeSegments, segments)
		}
	}

	var rO []TimeSegment
	if onlyOneUpdateTime {
		// ToDo: handle next update time not necessarily being on monday
		daysUntilMonday := (int(time.Monday) - int(timeSegments[0][0].Weekday()) + 7) % 7
		fmt.Printf("monday: %d | weekday: %d | modulod weekday: %d | daysUntilMonday: %d\n",
			int(time.Monday),
			int(timeSegments[0][0].Weekday()+7),
			int(timeSegments[0][0].Weekday()+7)%7,
			daysUntilMonday,
		)

		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}

		timeSegments[0][0] = timeSegments[0][0].AddDate(0, 0, daysUntilMonday)

		{
			rO = append(rO, TimeSegment{Type: "UpdateTimes", Times: timeSegments[0]})
		}

	} else {
		rO = append(rO, TimeSegment{Type: "UpdateTimes", Times: timeSegments[0]})
		rO = append(rO, TimeSegment{Type: "OperatingHoursMorning", Times: timeSegments[1]})
		rO = append(rO, TimeSegment{Type: "OperatingHoursAfternoon", Times: timeSegments[2]})
	}

	fmt.Println(rO)
	return rO
}

func main() {
	// Usual transcript
	transcript1 := `this message is updated at 7 30, 13 15 and 17 05 local time
	todays flight operating hours are from 7 30 till 12 15 local time and from 13 00 to 17 15 local time
	meiringen ctr and ALL tma sectors are active`

	// Over the weekend
	transcript2 := `Meiringen ctr and tma are not active
	expect ctr and tma meiringen to be active again next monday through 7 30 local time.
	if you hear this message on monday after 07 30 local time, contact...`

	fmt.Println(transcript1)
	ParseTimeSegments(transcript1)

	fmt.Println("")
	fmt.Println("-----")
	fmt.Println("")

	fmt.Println(transcript2)
	ParseTimeSegments(transcript2)
}
