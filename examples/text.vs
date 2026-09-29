// Text renderer regression / example for VScript 0.2.3

let fontPath = env("VSCRIPT_FONT", "");
if fontPath == "" && exists("/system/fonts/Roboto-Regular.ttf") {
    fontPath = "/system/fonts/Roboto-Regular.ttf";
} else if fontPath == "" && exists("/usr/share/fonts/fonts-go/Go-Regular.ttf") {
    fontPath = "/usr/share/fonts/fonts-go/Go-Regular.ttf";
}

let titleFont = null;
if fontPath != "" {
    titleFont = font(fontPath, 48);
}

video 640 360 10 4s design 640 360 {
    background("#08101a");
    rect(20, 20, 600, 320, "#102033", 20);

    // Built-in 5x7 renderer: size is now the approximate glyph height.
    text("BUILT IN TEXT", 320, 90, 42, "#ffffff");
    text("A B C 0123", 320, 150, 30, "#66ccff");

    // Native FreeType renderer when a font file is available.
    if titleFont != null {
        text("NATIVE FONT", 320, 220, 42, "#ffffff", titleFont);
        text("font size follows text() size", 320, 275, 24, "#a8b5c6", titleFont);
    } else {
        text("NATIVE FONT UNAVAILABLE", 320, 220, 26, "#ffcc66");
        text("set VSCRIPT_FONT to a TTF/OTF path", 320, 275, 18, "#a8b5c6");
    }

    circle(90, 310, 18, "#00d9ff");
    circle(550, 310, 18, "#ff006e");
}

export video "output/text-0.2.3.mp4" codec "libx264" bitrate 2500000;
