// siltide.  The behaviour every page shares: the recorded session player,
// the language switch and the colour mode switch.
// The hero is a session recorded with the interface itself: every line is
// the terminal's own output, replayed here as text rather than as a picture.
(function(){
  var screen = document.getElementById("demo-screen");
  var toggle = document.getElementById("demo-toggle");
  if (!screen || !toggle) { return; }
  var rows = [], frames = [], at = 0, timer = null, running = false;

  function label() {
    var zh = (document.documentElement.lang || "").indexOf("zh") === 0;
    toggle.textContent = toggle.getAttribute(zh ? "data-zh" : "data-en");
  }
  function draw(i) {
    var lines = frames[i].lines;
    for (var k in lines) { if (rows[k]) { rows[k].innerHTML = lines[k]; } }
  }
  function step() {
    draw(at);
    var hold = 1 + (frames[at].hold || 0);
    at = (at + 1) % frames.length;
    timer = setTimeout(step, hold * 100);
  }
  function play() {
    if (running) { return; }
    running = true;
    toggle.setAttribute("data-en", "pause");
    toggle.setAttribute("data-zh", "暂停");
    label();
    step();
  }
  function pause() {
    running = false;
    clearTimeout(timer);
    toggle.setAttribute("data-en", "play");
    toggle.setAttribute("data-zh", "播放");
    label();
  }
  toggle.addEventListener("click", function(){ if (running) { pause(); } else { play(); } });

  fetch("assets/demo.json").then(function(r){ return r.json(); }).then(function(reel){
    for (var i = 0; i < reel.rows; i++) {
      rows.push(screen.appendChild(document.createElement("div")));
    }
    frames = reel.frames;
    draw(0);
    var still = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (still) { pause(); } else { play(); }
  }).catch(function(){
    // No animation without the recording: the picture of it still works.
    var img = document.createElement("img");
    img.src = "assets/demo.gif";
    img.alt = "siltide moving through its tabs on a simulated fleet";
    img.style.width = "100%";
    screen.parentNode.replaceChild(img, screen);
    toggle.remove();
  });
})();

document.querySelectorAll(".install .tabbtn").forEach(function(b){
  b.addEventListener("click", function(){
    document.querySelectorAll(".install .tabbtn").forEach(function(x){ x.classList.remove("active"); });
    document.querySelectorAll(".install-panel").forEach(function(x){ x.classList.remove("active"); });
    b.classList.add("active");
    document.getElementById("p-" + b.dataset.p).classList.add("active");
  });
});
document.querySelectorAll(".copy").forEach(function(btn){
  btn.addEventListener("click", function(){
    var text = btn.parentElement.innerText.replace(/^copy\n?/, "");
    navigator.clipboard.writeText(text).then(function(){
      var was = btn.textContent; btn.textContent = "copied";
      setTimeout(function(){ btn.textContent = was; }, 1200);
    });
  });
});

// Bilingual toggle: every element carrying data-en/data-zh (plain text) or
// data-en-html/data-zh-html (the few with an inline link or code span)
// swaps on click, remembered in localStorage, defaulting to the browser's
// language on a first visit.
var SILTIDE_LANG_KEY = "siltide-lang";
function accelApplyLang(lang){
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  document.querySelectorAll("[data-en]").forEach(function(el){
    el.textContent = lang === "zh" ? el.getAttribute("data-zh") : el.getAttribute("data-en");
  });
  document.querySelectorAll("[data-en-html]").forEach(function(el){
    el.innerHTML = lang === "zh" ? el.getAttribute("data-zh-html") : el.getAttribute("data-en-html");
  });
  var btn = document.getElementById("lang-toggle");
  if (btn) btn.textContent = lang === "zh" ? "EN" : "中文";
  try { localStorage.setItem(SILTIDE_LANG_KEY, lang); } catch (e) {}
}
(function(){
  var start = "en";
  try { start = localStorage.getItem(SILTIDE_LANG_KEY) || ""; } catch (e) {}
  if (!start) {
    start = (navigator.language || "").toLowerCase().indexOf("zh") === 0 ? "zh" : "en";
  }
  accelApplyLang(start);
  var btn = document.getElementById("lang-toggle");
  if (btn) btn.addEventListener("click", function(){
    accelApplyLang(document.documentElement.lang === "zh-CN" ? "en" : "zh");
  });
})();

// Color mode: dark by default, light on request, remembered in
// localStorage, defaulting to the system preference on a first visit.
var SILTIDE_MODE_KEY = "siltide-mode";
function accelApplyMode(mode){
  if (mode === "light") { document.documentElement.setAttribute("data-mode", "light"); }
  else { document.documentElement.removeAttribute("data-mode"); }
  var btn = document.getElementById("mode-toggle");
  if (btn) btn.textContent = mode === "light" ? "dark" : "light";
  try { localStorage.setItem(SILTIDE_MODE_KEY, mode); } catch (e) {}
}
(function(){
  var start = "";
  try { start = localStorage.getItem(SILTIDE_MODE_KEY) || ""; } catch (e) {}
  if (!start) {
    start = (window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches) ? "light" : "dark";
  }
  accelApplyMode(start);
  var btn = document.getElementById("mode-toggle");
  if (btn) btn.addEventListener("click", function(){
    accelApplyMode(document.documentElement.getAttribute("data-mode") === "light" ? "dark" : "light");
  });
})();
