package coinank

import "os"

var TestApikey = os.Getenv("COINANK_API_KEY") // integration tests require COINANK_API_KEY
