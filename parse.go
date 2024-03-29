// The rbxdhist package is used to parse logs involving the deployment of Roblox
// builds.
package rbxdhist

import (
	"regexp"
	"time"

	"github.com/robloxapi/rbxver"
)

// Notes about build process (as deduced by examining the log file)
//
// When a job starts (New/Revert), it writes a message to the log file with the
// current time (time indicates when job starts). When the job finishes (Done)
// or fails (Error), it writes this status to the file. A job may be canceled,
// in which case no status is written. The Revert message indicates which
// version is reverted to.
//
// If multiple jobs start at the same time, they can inadvertently write to the
// file out of order (occurs on 2017/1/5). This is why status messages are
// separated from job messages. There is also at least one status message that
// does not seem to match any job message.
//
// There is one case of an irregular job message being emitted (2012/6/29).
// Because the message occurs only once, it is not included as a standard
// message, instead being parsed as a raw token.
//
// Job messages have changed in style over time. To justify the complex regexp,
// these changes are documented below (\n: newline, \s: trailing space).
const parserGrammar = `` +
	// Original style message:
	//     New Build version-0123456789abcdef at 1/2/2006 3:04:05 PM...\s
	//     Revert Build version-0123456789abcdef at 1/2/2006 3:04:05 PM...\s
	// Version addition (2011/1/6):
	//     New Build version-0123456789abcdef at 1/2/2006 3:04:05 PM, file verion: 0, 123, 1, 12345...
	// Spelling correction (2012/6/28):
	//     New Build version-0123456789abcdef at 1/2/2006 3:04:05 PM, file version: 0, 123, 1, 12345...
	// Newline prefix (2012/9/6):
	//     \nNew Build version-0123456789abcdef at 1/2/2006 3:04:05 PM, file version: 0, 123, 1, 12345...
	//     \nRevert Build version-0123456789abcdef at 1/2/2006 3:04:05 PM...
	// Git hash (2021/5/24):
	//     \nNew Build version-0123456789abcdef at 1/2/2006 3:04:05 PM, file version: 0, 123, 1, 12345, git hash: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa ...
	`(?:(?:^|\r?\n)?(New|Revert) (\w+) (.*?) at (\d{1,2}/\d{1,2}/\d{3,4} \d{1,2}:\d{2}:\d{2} (?:A|P)M)(?:, file vers?ion: (\d+, \d+, \d+, \d+))?(?:, git hash: ([0-9a-fA-F]+) )?... ?)` +
	// Status (unchanged):
	//     Done!\n
	//     Error!\n
	`|(?:(Done|Error)!\r?\n)`

var streamParser = regexp.MustCompile(parserGrammar)

// Specifies the location to use when parsing dates. If nil, the assumed
// location will be "America/Los_Angeles", which is loaded according to
// time.LoadLocation. If an error occurs, then times will be parsed as local.
var TimeZone *time.Location = nil

// Lex processes a stream of bytes into a stream of tokens.
func Lex(b []byte) (s Stream) {
	zonePST := TimeZone
	if zonePST == nil {
		var err error
		zonePST, err = time.LoadLocation("America/Los_Angeles")
		if err != nil {
			zonePST, _ = time.LoadLocation("Local")
		}
	}
	for i := 0; i < len(b); {
		r := streamParser.FindSubmatchIndex(b[i:])
		if len(r) == 0 || r[1] < 0 {
			// EOF
			if i < len(s) {
				raw := Raw(b[i:])
				s = append(s, &raw)
			}
			break
		}
		if r[0] > 0 {
			// There is some unparsed text between known messages.
			raw := Raw(b[i : i+r[0]])
			s = append(s, &raw)
		}
		if r[2] >= 0 {
			// Job message.
			job := Job{
				Action: string(b[i+r[2] : i+r[3]]),
				Build:  string(b[i+r[4] : i+r[5]]),
				GUID:   string(b[i+r[6] : i+r[7]]),
			}
			const dateLayout = "1/2/2006 3:04:05 PM"
			var err error
			if job.Time, err = time.ParseInLocation(dateLayout, string(b[i+r[8]:i+r[9]]), zonePST); err != nil {
				goto parseRaw
			}
			if r[10] >= 0 {
				if job.Version = rbxver.Parse(string(b[i+r[10]:i+r[11]]), rbxver.Any); job.Version.Format == rbxver.Any {
					goto parseRaw
				}
			}
			if r[12] >= 0 {
				job.GitHash = string(b[i+r[12] : i+r[13]])
			}
			s = append(s, &job)
		} else if r[14] >= 0 {
			// Status message.
			status := Status(b[i+r[14] : i+r[15]])
			s = append(s, &status)
		}
		goto next
	parseRaw:
		// Reparse as raw text.
		if r[0] > 0 {
			// Append to previous parseRaw text.
			raw := *(s[len(s)-1].(*Raw)) + Raw(b[i+r[0]:i+r[1]])
			s[len(s)-1] = &raw
		} else {
			raw := Raw(b[i : i+r[0]])
			s = append(s, &raw)
		}
	next:
		i += r[1]
	}
	return s
}
