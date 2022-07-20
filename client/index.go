package main

const index = `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8"/>
    <title>goMarkableStream</title>
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
.orientation {
  position: absolute;
  top: 10px;
  left: 10px;
}
.orientation button {
  background-color: #04AA6D;
  color: white;
  padding: 12px 20px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  width: 100%;
  margin-bottom: 10px;
}
</style>
  </head>
  <body>
    <div class="container center">
    {{ if not .AutoRotate }}
    <div class="orientation">
      <iframe name="dummyframe" id="dummyframe" style="display: none;"></iframe>
      <form action="/orientation" method="GET" target="dummyframe">
        <input type="hidden" name="orientation" value="portrait" />
        <button>portrait</button>
      </form>
      <form action="/orientation" method="GET" target="dummyframe">
        <input type="hidden" name="orientation" value="landscape" />
        <button>landscape</button>
      </form>
    </div>
    {{end}}
    <a href="/screenshot" download>
      <img src="/video">
      </a>
    </div>
  </body>
</html>
`
