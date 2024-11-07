package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

type AirspaceStatus struct {
	Areas          []Area      `json:"areas"`
	NextUpdate     time.Time   `json:"nextUpdate"`
	OperatingHours []time.Time `json:"operatingHours"`
}

type Area struct {
	Index  int  `json:"index"`
	Status bool `json:"status"`
}

type TimeSegment struct {
	Type  string
	Times []time.Time
}

// ---- Testing -----
type HxAreaTestJson struct {
	HxArea                      string            `json:"hx_area"`
	Transcript                  string            `json:"transcript"`
	AdditionalNote              string            `json:"additionalNote"`
	ExpectedVerdict             string            `json:"expectedVerdict"`
	ExpectedHxAreasActiveStatus []map[string]bool `json:"expectedHxAreasActiveStatus"`
	TestingTimeAndDate          time.Time         `json:"testingTimeAndDate"`
	ExpectedNextAction          time.Time         `json:"expectedNextAction"`
}

// ------------------

// parseTranscript parses the provided transcript and extracts the airspace status
func parseAirspaceStates(transcript string) AirspaceStatus {
	// Default areas
	areas := []Area{
		{0, false}, // CTR
		{1, false}, // TMA x
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
	hasAreNotActive := strings.Contains(transcript, "are not active")

	everyTmaTargeted := strings.Contains(splitActive[0], "all tma")

	fmt.Printf(
		"[i] canBeActivated: %t | hasAreNotActive: %t | everyTmaTargeted: %t | hasMultipleCtrSubstrings: %t\n",
		canBeActivated,
		hasAreNotActive,
		everyTmaTargeted,
		hasMultipleCtrSubstrings,
	)

	if !canBeActivated && !everyTmaTargeted && !hasAreNotActive {
		// CTR and specific TMAs are active
		activeTmas := regexp.MustCompile(`\d`).FindAllString(splitActive[0], -1)
		fmt.Printf("[i] Len of activeTmas (%v): %d\n", activeTmas, len(activeTmas))

		for i := range activeTmas {
			areas[i+1].Status = true
		}

		// CTR
		areas[0].Status = true
	} else if !hasAreNotActive && everyTmaTargeted {
		// Eveything is active
		for i := range areas {
			areas[i].Status = true
		}
	} else if hasAreNotActive {
		// Everything is inactive, therefore preserve defaults
	}

	// Return the parsed data
	return AirspaceStatus{
		Areas:      areas,
		NextUpdate: time.Unix(0, 0),
	}
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

// Extract time segments; Next updates and flight operating hours
func parseTimeSegments(transcript string) []TimeSegment {
	patternTimeSegments := `\d{1,2}[: ]\d{2}`

	// Split all time segments by the "local time" substring.
	// Segment 1: Message update times,
	// Segment 2: Flight operating hours morning,
	// Segment 3: Flight operating hours evening,
	// Format: [[t1, t2], [t1, t2], [t1, t2]]
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
		trimmed := strings.TrimSpace(split)
		if onlyOneUpdateTime && i == 1 {
			continue
		}

		if regexp.MustCompile(`\d`).MatchString(trimmed) {
			re := regexp.MustCompile(patternTimeSegments)
			matches := re.FindAllStringSubmatch(trimmed, -1)

			// Parse each time segment within local time segment, but only if we have any matches at alls
			if len(matches) < 1 {
				continue
			}

			var segments []time.Time
			for _, match := range matches {
				replacedString := strings.Replace(match[0], " ", ":", 1)
				convertedTime, err := parseTimeToCurrentDate(replacedString)
				if err != nil {
					panic(err)
				}

				segments = append(segments, convertedTime)
			}

			timeSegments = append(timeSegments, segments)
		}
	}

	var rO []TimeSegment
	if onlyOneUpdateTime {
		// If yes, then the update time will be on a future day
		// ToDo: handle next update time not necessarily being on monday
		daysUntilMonday := (int(time.Monday) - int(timeSegments[0][0].Weekday()) + 7) % 7

		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}

		timeSegments[0][0] = timeSegments[0][0].AddDate(0, 0, daysUntilMonday)
	} else {
		var operatingHours []time.Time
		for i := range timeSegments {
			// Skip first timeSegment, the update times
			if i > 0 {
				operatingHours = append(operatingHours, timeSegments[i]...)
			}
		}

		rO = append(rO, TimeSegment{Type: "OperatingHours", Times: operatingHours})
	}

	rO = append(rO, TimeSegment{Type: "UpdateTimes", Times: timeSegments[0]})
	return rO
}

func ParseTranscript(transcript string) AirspaceStatus {
	var timeSegments []TimeSegment
	var airspaceState AirspaceStatus

	timeSegments = parseTimeSegments(transcript)
	airspaceState = parseAirspaceStates(transcript)

	// Assign time segments
	var updateTimeTimeSegment TimeSegment
	var operatingHoursTimeSegment TimeSegment
	for _, segment := range timeSegments {
		if segment.Type == "UpdateTimes" {
			updateTimeTimeSegment = segment
		} else if segment.Type == "OperatingHours" {
			operatingHoursTimeSegment = segment
		}
	}

	var nextUpdateTime time.Time
	now := time.Now()
	for _, segment := range updateTimeTimeSegment.Times {
		if now.Before(segment) {
			nextUpdateTime = segment
		}
	}

	airspaceState.NextUpdate = nextUpdateTime
	airspaceState.OperatingHours = operatingHoursTimeSegment.Times

	o, _ := json.Marshal(airspaceState)
	color.Yellow(fmt.Sprintf("%v", string(o)))

	return airspaceState
}

func main() {
	jsonFile, err := os.Open("../test-transcripts.json")
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	var hxTestStatuses []HxAreaTestJson
	j, _ := io.ReadAll(jsonFile)
	if err := json.Unmarshal(j, &hxTestStatuses); err != nil {
		panic(err)
	}

	var mismatches int
	for i, hxTestStatus := range hxTestStatuses {
		fmt.Printf("\n---------------------[%d]---------------------\n", i+1)
		fmt.Printf("- Transcript: %s\n", hxTestStatus.Transcript)
		fmt.Printf("- Testing time: %s\n", hxTestStatus.TestingTimeAndDate)
		fmt.Printf("- Expected next action: %s\n", hxTestStatus.ExpectedNextAction)
		fmt.Printf("- Expected HX area statuses: %v\n", hxTestStatus.ExpectedHxAreasActiveStatus)

		airspaceState := ParseTranscript(hxTestStatus.Transcript)

		// Verify
		for _, area := range airspaceState.Areas {
			// Roundabout way of acquiring our expected HS status
			var expectedAreaStatus bool
			for _, m := range hxTestStatus.ExpectedHxAreasActiveStatus {
				if val, exists := m[strconv.Itoa(area.Index)]; exists {
					expectedAreaStatus = val
					break
				}
			}

			var verdict string
			var verdictColor *color.Color
			if expectedAreaStatus == area.Status {
				verdictColor = color.New(color.FgGreen)
				verdict = "Match"
			} else {
				mismatches += 1
				verdictColor = color.New(color.FgRed)
				verdict = "Mismatch"
			}

			fmt.Printf(
				"Area %d: Parsed (%v) vs expected (%v): ",
				area.Index,
				area.Status,
				expectedAreaStatus,
			)

			verdictColor.Println(verdict)
		}
	}

	if mismatches > 0 {
		panic(fmt.Sprintf("There have been %d mismtaches", mismatches))
	}
}
