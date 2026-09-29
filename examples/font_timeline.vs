let fontPath = env("VSCRIPT_FONT", "/usr/share/fonts/fonts-go/Go-Bold.ttf");
let f = font(fontPath, 32);
let tl = timeline(5);
tl.keyframe("x", 0, -120);
tl.keyframe("x", 2.5, 240);
tl.keyframe("x", 5, 500);

video 1280 720 30fps 5s design 640 360 {
    background("#09111c");
    let x = tl.value("x", t);
    rect(x, 135, 260, 90, "#1f5d94", 24);
    text("FONT + TIMELINE", 320, 180, 32, "#ffffff", f);
}
export video "output/font-timeline.mp4" codec "libx264" bitrate 4000000;
