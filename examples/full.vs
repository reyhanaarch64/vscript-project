let title = "vscript";
let subtitle = "procedural video";
let baseY = 90;

function wave(center, amplitude, speed) {
    return center + sin(t * speed) * amplitude;
}

video 640 360 30 3s {
    background("#080d18");

    rect(24, 24, 592, 312, "#101a2a", 28);
    polygon([54, 290, 160, 58, 270, 290], "#18395e");
    polygon([370, 290, 480, 54, 590, 290], "#16324f");

    circle(320, wave(baseY, 26, 2.8) + 80, 38, "#55a9ff");
    circle(320, wave(baseY, 26, 2.8) + 80, 60, "#204a76");

    text(title, 320, 185, 10, "#ffffff");
    text(subtitle, 320, 225, 5, "#9fc9ef");

    line(76, 270, 564, 270, "#294766", 3);
    filter("contrast", 1.08);
    filter("brightness", 4);
    filter("vignette", 0.34);
}

export video "output/vscript-full.mp4" codec "libx264" bitrate 2200000;
