video 320 240 15 1.2s {
    background("#202020");
    rect(20, 20, 280, 200, "#e63946", 22);
    circle(160, 120, 70, "#f1fa8c");
    filter("brightness", 12);
    filter("contrast", 1.15);
    filter("sepia");
    filter("scanlines", 0.15);
}

export video "output/vscript-filters.mp4" codec "libx264" bitrate 1200000;
