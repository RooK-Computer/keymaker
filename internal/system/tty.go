package system

import (
    "fmt"
    "golang.org/x/sys/unix"
)

// KD console modes from linux/kd.h
const (
    kdText     = 0x00
    kdGraphics = 0x01
    kdSetMode  = 0x4B3A // KDSETMODE ioctl
)

// SetGraphicsMode switches the active console to graphics mode to suppress the hardware cursor.
// It targets /dev/tty0, the current active virtual terminal.
func SetGraphicsMode() error {
    fd, err := unix.Open("/dev/tty0", unix.O_RDONLY|unix.O_NONBLOCK, 0)
    if err != nil {
        return fmt.Errorf("open /dev/tty0: %w", err)
    }
    defer unix.Close(fd)
    if err := unix.IoctlSetInt(fd, kdSetMode, kdGraphics); err != nil {
        return fmt.Errorf("ioctl KDSETMODE KD_GRAPHICS: %w", err)
    }
    return nil
}

// RestoreTextMode restores the console to text mode so cursor and normal console return.
func RestoreTextMode() error {
    fd, err := unix.Open("/dev/tty0", unix.O_RDONLY|unix.O_NONBLOCK, 0)
    if err != nil {
        return fmt.Errorf("open /dev/tty0: %w", err)
    }
    defer unix.Close(fd)
    if err := unix.IoctlSetInt(fd, kdSetMode, kdText); err != nil {
        return fmt.Errorf("ioctl KDSETMODE KD_TEXT: %w", err)
    }
    return nil
}
