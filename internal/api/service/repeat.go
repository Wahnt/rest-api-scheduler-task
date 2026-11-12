package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const DateFormat = "20060102"

// NextDate calculate next date with rule repeat
// now - now, dstart - first date, repeat - rule
// return date > now format(20060102)
func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	if strings.TrimSpace(repeat) == "" {
		return "", fmt.Errorf("repeat rule is empty")
	}

	start, err := time.Parse(DateFormat, dstart)
	if err != nil {
		return "", fmt.Errorf("ivalid dstart %q: %w", dstart, err)
	}

	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty repeat rule")
	}

	cur := start

	for {
		var next time.Time

		switch parts[0] {
		case "d":
			next, err = nextDay(parts[1:], cur)
		case "w":
			next, err = nextWeekday(parts[1:], cur)
		case "m":
			next, err = nextMonthDay(parts[1:], cur)
		case "y":
			next, err = nextYear(parts[1:], cur)
		default:
			return "", fmt.Errorf("unknown repeat rule: %q", parts[0])
		}
		if err != nil {
			return "", err
		}

		cur = next
		if cur.After(nowDate) {
			return cur.Format(DateFormat), nil
		}
	}
}

// nextDay shifts the date forward by N days
func nextDay(args []string, cur time.Time) (time.Time, error) {
	if len(args) != 1 {
		return time.Time{}, fmt.Errorf("d rule expects 1 argument")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("ivalid number %q", args[0])
	}
	if n < 1 || n > 400 {
		return time.Time{}, fmt.Errorf("number must be 1..400, got %d:", n)
	}
	return cur.AddDate(0, 0, n), nil
}

// nextYear shifts the date forward by 1 year
func nextYear(args []string, cur time.Time) (time.Time, error) {
	if len(args) != 0 {
		return time.Time{}, fmt.Errorf("y rule takes no arguments")
	}
	return cur.AddDate(1, 0, 0), nil
}

// nextWeekday finds the nearest day that falls on the days of the week
func nextWeekday(args []string, cur time.Time) (time.Time, error) {
	if len(args) != 1 {
		return time.Time{}, fmt.Errorf("w rule xpects 1 argument")
	}

	allowed := make(map[int]bool) // valid week days
	for _, part := range strings.Split(args[0], ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > 7 {
			return time.Time{}, fmt.Errorf("invalid weekday %q", part)
		}
		allowed[n] = true
	}
	if len(allowed) == 0 {
		return time.Time{}, fmt.Errorf("empty weekday list")
	}

	for i := 1; i <= 7; i++ {
		c := cur.AddDate(0, 0, i)
		if allowed[weekdayNumber(c)] {
			return c, nil
		}
	}
	return time.Time{}, fmt.Errorf("no matching weekday")
}

// weekdayNumber change(conversion) time.Weekday
// default time.Sunday = 0
func weekdayNumber(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// nextMonthDay find the nearest day that falls on the opposite side of the month
func nextMonthDay(args []string, cur time.Time) (time.Time, error) {
	if len(args) < 1 || len(args) > 2 {
		return time.Time{}, fmt.Errorf("rule expects 1 or 2 arguments")
	}

	allowedDays, err := parseMonthDays(args[0])
	if err != nil {
		return time.Time{}, err
	}

	var allowedMonths map[int]bool
	if len(args) == 2 {
		allowedMonths, err = parseMonths(args[1])
		if err != nil {
			return time.Time{}, err
		}
	}
	y, m := cur.Year(), int(cur.Month())
	for i := 0; i < 2400; i++ {
		if allowedMonths == nil || allowedMonths[m] {
			last := lastDayOfMonth(cur)
			daysInMonth := time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
			_ = last
			for _, d := range allowedDays {
				day := d
				if d < 0 {
					day = daysInMonth + d + 1
				}
				if day < 1 || day > daysInMonth {
					continue
				}
				target := time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
				if target.After(cur) {
					return target, nil
				}
			}
		}
		m++
		if m > 12 {
			m = 1
			y++
		}
	}
	return time.Time{}, fmt.Errorf("no matching date within 2 years")
}

// matchingMonthDay check day of month and does the day fit into the array
func matchingMonthDay(t time.Time, allowedDays []int) bool {
	day := t.Day()
	last := lastDayOfMonth(t)
	for _, d := range allowedDays {
		switch {
		case d > 0 && d == day:
			return true
		case d == -1 && day == last:
			return true
		case d == -2 && day == last-1:
			return true
		}
	}
	return false
}

func parseMonthDays(s string) ([]int, error) {
	var days []int

	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid day %q: ", part)
		}
		if n == -1 || n == -2 {
			days = append(days, n)
		} else if n >= 1 && n <= 31 {
			days = append(days, n)
		} else {
			return nil, fmt.Errorf("day must be 1..31, -1, -2, got %d", n)
		}
	}
	if len(days) == 0 {
		return nil, fmt.Errorf("empty day list")
	}
	return days, nil
}

func parseMonths(s string) (map[int]bool, error) {
	months := make(map[int]bool)
	for _, part := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > 12 {
			return nil, fmt.Errorf("invalid month %q", part)
		}
		months[n] = true
	}
	if len(months) == 0 {
		return nil, fmt.Errorf("empty month list")
	}
	return months, nil
}

func lastDayOfMonth(t time.Time) int {
	return t.AddDate(0, 1, -1).Day()
}
