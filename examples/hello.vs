let title = "vscript";
let amount = 45;

function wave(base, speed) {
    return base + sin(t * speed) * amount;
}

video 640 360 30 3s {
    background("#080d18");
    rect(28, 28, 584, 304, "#111a2b", 24);
    circle(320, wave(170, 3), 44, "#55a9ff");
    circle(320, wave(170, 3), 82, "#1f4b76");
    line(54, 262, 586, 262, "#274667", 3);
    text(title, 320, 305, 8, "#ffffff");
    filter("vignette", 0.42);
}

export video "output/vscript-demo.mp4" codec "libx264" bitrate 2800000;
