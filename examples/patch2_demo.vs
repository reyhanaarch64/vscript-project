// vscript 0.2.0 - demo fitur patch 2
// 1280x720 dengan design canvas 480x270.

let W = 1280;
let H = 720;
let FPS = 15;
let DURATION = 6;
let out = "output/patch2-demo.mp4";
let fontPath = env("VSCRIPT_FONT", "");

if fontPath == "" && exists("/system/fonts/Roboto-Bold.ttf") {
    fontPath = "/system/fonts/Roboto-Bold.ttf";
} else if fontPath == "" && exists("/usr/share/fonts/fonts-go/Go-Bold.ttf") {
    fontPath = "/usr/share/fonts/fonts-go/Go-Bold.ttf";
} else {
    fontPath = "";
}

let uiFont = null;
if fontPath != "" {
    uiFont = font(fontPath, 28);
}

let motion = timeline(6);
motion.keyframe("x", 0, -120);
motion.keyframe("x", 1.4, 240);
motion.keyframe("x", 3.2, 360);
motion.keyframe("x", 5.8, 520);
motion.keyframe("card", 0, {x: -70, y: 138});
motion.keyframe("card", 2, {x: 60, y: 138});
motion.keyframe("card", 4, {x: 145, y: 112});
motion.keyframe("card", 6, {x: 225, y: 138});

function titleText() {
    if (t < 2) {
        return "REHAN";
    } else if (t < 4) {
        return "PROGRAMMER";
    } else {
        return "VSCRIPT";
    }
}

function titleColor() {
    if (t < 2) {
        return "#ffffff";
    } else if (t < 4) {
        return "#66ccff";
    } else {
        return "#ffcc66";
    }
}

video W H FPS DURATION design 480 270 {
    background("#081018");

    let card = motion.value("card", t);
    let x = motion.value("x", t);
    let pulse = 18 + sin(t * 5) * 3;

    rect(18, 18, 444, 234, "#0f1b2a", 22);
    rect(card.x, card.y, 250, 72, "#16304b", 20);
    circle(402, 60, pulse, "#2b78d0");
    polygon([x, 224, x + 22, 188, x + 44, 224], "#ffcc66");
    line(40, 224, 440, 224, "#244863", 3);

    if uiFont != null {
        text(titleText(), 240, 70, 28, titleColor(), uiFont);
        text("general-purpose + procedural video", 240, 104, 12, "#a7bfd8", uiFont);
        text("timeline / keyframe / if-else / font / filters", 240, 236, 10, "#7f9ab5", uiFont);
    } else {
        text(titleText(), 240, 70, 10, titleColor());
        text("timeline / keyframe / if-else / filters", 240, 104, 6, "#a7bfd8");
    }

    image(50, 140, 96, 54, "assets/demo-pattern.png");

    if t > 4.8 {
        filter("contrast", 1.08);
        filter("brightness", 6);
        filter("vignette", 0.18);
    }
}

export video out codec "libx264" bitrate 5000000;
