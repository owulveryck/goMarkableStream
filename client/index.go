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
</style>
  </head>
  <body>
    <div class="container center">
    <a href="/screenshot" download>
      <img src="/video">
      </a>
    </div>
  </body>
</html>
`
