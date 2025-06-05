package gobookmarks

import "os"

var UseCssColumns = os.Getenv("GBM_CSS_COLUMNS") != ""
