package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type AirspaceStatus struct {
	Areas      []Area    `json:"areas"`
	NextUpdate time.Time `json:"nextUpdate"`
}

type Area struct {
	Index  int  `json:"index"`
	Status bool `json:"status"`
}

type TimeSegment struct {
	Type  string
	Times []time.Time
}

// parseTranscript parses the provided transcript and extracts the airspace status
func parseAirspaceStates(transcript string) (AirspaceStatus, error) {
	// Default areas
	areas := []Area{
		{0, false}, // CTR
		{1, false}, // TMA n
		{2, false},
		{3, false},
		{4, false},
		{5, false},
		{6, false},
	}

	transcript = strings.ToLower(transcript)

	// If true, then transcript is from a time outside flight operating hours
	// As such, all mentioned sectors are inactive
	canBeActivated := strings.Contains(transcript, "can be")

	// If true, then likely no areas will be active (weekend transcript)
	hasMultipleCtrSubstrings := strings.Count(transcript, "ctr") > 1
	var ctrSubstringIndex int
	if hasMultipleCtrSubstrings {
		ctrSubstringIndex = 0
	} else {
		ctrSubstringIndex = 1
	}

	// First split by CTR, then by keyword "active"
	splitCtr := strings.Split(transcript, "ctr")
	splitActive := strings.Split(splitCtr[ctrSubstringIndex], "active")
	fmt.Printf("[i] splitCtr (%d): %v\n[i] splitActive (%d): %v\n", len(splitCtr), splitCtr, len(splitActive), splitActive)

	// If contained in the first split segment, then no areas are active
	hasAreNotActive := strings.Contains(splitCtr[0], "are not active")

	everyTmaTargeted := strings.Contains(splitActive[0], "all tma")

	fmt.Printf(
		"[i] canBeActivated: %t | hasAreNotActive: %t | everyTmaTargeted: %t | hasMultipleCtrSubstrings: %t\n",
		canBeActivated,
		hasAreNotActive,
		everyTmaTargeted,
		hasMultipleCtrSubstrings,
	)

	if !canBeActivated && !everyTmaTargeted {
		// CTR and specific TMAs are active
		activeTmas := regexp.MustCompile(`\d`).FindAllString(splitActive[0], -1)
		fmt.Printf("[i] Len of activeTmas (%v): %d\n", activeTmas, len(activeTmas))

		for i := range activeTmas {
			areas[i].Status = true
		}

		// CTR
		areas[0].Status = true
	} else if !hasAreNotActive && everyTmaTargeted {
		// Eveything is active
		for i := range areas {
			areas[i].Status = true
		}
	} else if hasAreNotActive && everyTmaTargeted {
		// Everything is inactive, therefore preserve defaults
	}

	// Return the parsed data
	return AirspaceStatus{
		Areas:      areas,
		NextUpdate: time.Unix(0, 0),
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
	var transcripts []string

	// Usual transcript
	transcripts = append(transcripts, `this message is updated at 7 30, 13 15 and 17 05 local time
	todays flight operating hours are from 7 30 till 12 15 local time and from 13 00 to 17 15 local time
	meiringen ctr and ALL tma sectors are active`)

	// Over the weekend
	transcripts = append(transcripts, `Meiringen ctr and tma are not active
	expect ctr and tma meiringen to be active again next monday through 7 30 local time.
	if you hear this message on monday after 07 30 local time, contact...`)

	// Usual transcript, partially active TMAs
	transcripts = append(transcripts, `this message is updated at 7 30, 13 15 and 17 05 local time
	todays flight operating hours are from 7 30 til 18 15 local time
	meiringen ctr and tma 1, 2 and 3 are active
	tma sectors 4, 5 and 6 remain deactivated until the next update`)

	var timeSegments []TimeSegment
	var airspaceState AirspaceStatus

	for i, transcript := range transcripts {
		fmt.Printf("\n---------------------[%d]---------------------\n", i+1)
		fmt.Printf("[i] %s\n", transcript)

		timeSegments = ParseTimeSegments(transcript)
		airspaceState, _ = parseAirspaceStates(transcript)

		var nextUpdateTime time.Time
		now := time.Now()
		for _, timeSegment := range timeSegments[0].Times {
			if now.Before(timeSegment) {
				nextUpdateTime = timeSegment
			}
		}

		airspaceState.NextUpdate = nextUpdateTime

		out, _ := json.Marshal(airspaceState)
		fmt.Println(string(out))
	}
}
