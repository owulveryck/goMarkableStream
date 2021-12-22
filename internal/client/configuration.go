package client

import (
	"fmt"
	"html/template"
	"image"
	"net/http"
)

// Configuration of the client
type Configuration struct {
	ServerAddr            string `env:"RK_SERVER_ADDR,default=remarkable:2000"`
	BindAddr              string `env:"RK_CLIENT_BIND_ADDR,default=:8080"`
	AutoRotate            bool   `env:"RK_CLIENT_AUTOROTATE,default=true"`
	ScreenShotDest        string `env:"RK_CLIENT_SCREENSHOT_DEST,default=."`
	PaperTexture          string `env:"RK_CLIENT_PAPER_TEXTURE"`
	Highlight             bool   `env:"RK_CLIENT_HIGHLIGHT,default=false"`
	Colorize              bool   `env:"RK_CLIENT_COLORIZE,default=true"`
	paperTextureLandscape *image.Gray
	paperTexturePortrait  *image.Gray
}

func (c *Configuration) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("form").Parse(tmpl))
	switch r.Method {
	case "GET":
		err := t.Execute(w, c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "POST":
		// Call ParseForm() to parse the raw query and update r.PostForm and r.Form.
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		autorotate := r.FormValue("autorotate")
		c.AutoRotate = false
		if autorotate == "on" {
			c.AutoRotate = true
		}
		colorize := r.FormValue("colorize")
		c.Colorize = false
		if colorize == "on" {
			c.Colorize = true
		}
		err := t.Execute(w, c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "method not implemented", http.StatusNotImplemented)
		return
	}
}

const tmpl = `<!DOCTYPE html>
<html>

<head>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta charset="UTF-8" />
  <style>

	html, body {
		height: 100%;
		margin: 0;
		background-color: gray;
	}
	.container {
		margin: 0;
		display: block;
		height: 100%;
	}
	
	.center {
		margin: auto;
	}
	.container img {
		display: block;
		margin-left: auto;
		margin-right: auto;
		max-height: 100%;
		max-width: 100%;
	}

    .switch {
      position: relative;
      display: inline-block;
      width: 40px;
      height: 20px;
    }

    .switch input {
      opacity: 0;
      width: 0;
      height: 0;
    }

    .slider {
      position: absolute;
      cursor: pointer;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: #ccc;
      -webkit-transition: .4s;
      transition: .4s;
    }

    .slider:before {
      position: absolute;
      content: "";
      height: 12px;
      width: 12px;
      left: 4px;
      bottom: 4px;
      background-color: white;
      -webkit-transition: .4s;
      transition: .4s;
    }

    input:checked+.slider {
      background-color: #2196F3;
    }

    input:focus+.slider {
      box-shadow: 0 0 1px #2196F3;
    }

    input:checked+.slider:before {
      -webkit-transform: translateX(18px);
      -ms-transform: translateX(18px);
      transform: translateX(18px);
    }

    /* Rounded sliders */
    .slider.round {
      border-radius: 34px;
    }

    .slider.round:before {
      border-radius: 50%;
    }

    /* Style inputs, select elements and textareas */
    input[type=text],
    select,
    textarea {
      width: 100%;
      padding: 12px;
      border: 1px solid #ccc;
      border-radius: 4px;
      box-sizing: border-box;
      resize: vertical;
    }

    /* Style the label to display next to the inputs */
    label {
      display: inline-block;
    }

    /* Style the submit button */
    input[type=submit] {
      background-color: #04AA6D;
      color: white;
      padding: 12px 20px;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      float: right;
    }

    /* Style the container */
    .container {
      border-radius: 5px;
      background-color: #f2f2f2;
      padding: 20px;
    }

    /* Floating column for labels: 25% width */
    .col-25 {
      float: left;
      width: 25%;
      margin-top: 6px;
    }

    /* Floating column for inputs: 75% width */
    .col-75 {
      float: left;
      width: 75%;
      margin-top: 6px;
    }

    /* Clear floats after the columns */
    .row:after {
      content: "";
      display: table;
      clear: both;
    }

    /* Responsive layout - when the screen is less than 600px wide, make the two columns stack on top of each other instead of next to each other */
    @media screen and (max-width: 600px) {

      .col-25,
      .col-75,
      input[type=submit] {
        width: 100%;
        margin-top: 0;
      }
    }
  </style>
</head>

<body>
  <div>
    <div class="container">
	<iframe name="dummyframe" id="dummyframe" style="display: none;"></iframe>

      <form action="/conf" method="POST" target="dummyframe">
        <div class="row">
          <div class="col-25">
            <label for="lautorate">auto rotate</label>
          </div>
          <div class="col-25">

            <label class="switch">
              <input name="autorotate" type="checkbox" {{ if .AutoRotate }}checked{{end}}>
              <span class="slider round"></span>
            </label>
          </div>
        </div>
        <div class="row">
          <div class="col-25">
            <label for="lcolorize">colorize</label>
          </div>
          <div class="col-25">
            <label class="switch">
              <input name="colorize" type="checkbox" {{ if .Colorize }}checked{{end}}>
              <span class="slider round"></span>
            </label>
          </div>
        </div>
        <div class="row">
          <div class="col-25">
          <input type="submit" value="Submit">
        </div>
        </div>
      </form>
    </div>
	<div>
	<div class="container center">
    <a href="/screenshot" download="">
      <img src="/video" height="450">
      </a>
    </div>
    </div>
</body>

</html>
`
