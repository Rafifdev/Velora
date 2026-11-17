package tray

import (
	"sync"

	"fyne.io/systray"
)

type Controller struct {
	iconBytes []byte
	onOpen    func()
	onStopJam func()
	onQuit    func()

	mu      sync.Mutex
	started bool
	ready   bool
	running bool
	url     string

	mStatus  *systray.MenuItem
	mOpen    *systray.MenuItem
	mStopJam *systray.MenuItem
	mQuit    *systray.MenuItem
}

func NewController(iconBytes []byte, onOpen, onStopJam, onQuit func()) *Controller {
	return &Controller{
		iconBytes: iconBytes,
		onOpen:    onOpen,
		onStopJam: onStopJam,
		onQuit:    onQuit,
	}
}

func (c *Controller) EnsureStarted(jamRunning bool, remoteURL string) {
	c.mu.Lock()
	c.running = jamRunning
	c.url = remoteURL
	alreadyStarted := c.started
	c.started = true
	c.mu.Unlock()

	if alreadyStarted {
		c.UpdateJamState(jamRunning, remoteURL)
		return
	}

	go systray.Run(c.onReady, func() {})
}

func (c *Controller) UpdateJamState(jamRunning bool, remoteURL string) {
	c.mu.Lock()
	c.running = jamRunning
	c.url = remoteURL
	ready := c.ready
	mStatus := c.mStatus
	mStopJam := c.mStopJam
	c.mu.Unlock()

	if !ready {
		return
	}

	if jamRunning {
		systray.SetTooltip("Velora Jam Aktif • " + remoteURL)
		if mStatus != nil {
			mStatus.SetTitle("Velora Jam: Aktif")
		}
		if mStopJam != nil {
			mStopJam.Show()
		}
	} else {
		systray.SetTooltip("Velora (Jam Nonaktif)")
		if mStatus != nil {
			mStatus.SetTitle("Velora Jam: Nonaktif")
		}
		if mStopJam != nil {
			mStopJam.Hide()
		}
	}
}

func (c *Controller) Shutdown() {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()
	if started {
		systray.Quit()
	}
}

func (c *Controller) onReady() {
	if len(c.iconBytes) > 0 {
		systray.SetIcon(c.iconBytes)
	}
	systray.SetTitle("Velora")
	systray.SetOnTapped(func() {
		if c.onOpen != nil {
			c.onOpen()
		}
	})

	mStatus := systray.AddMenuItem("Velora Jam: Aktif", "Status server Velora Jam")
	mStatus.Disable()
	systray.AddSeparator()
	mOpen := systray.AddMenuItem("Buka Velora", "Tampilkan jendela Velora")
	mStopJam := systray.AddMenuItem("Matikan Velora Jam", "Hentikan server remote Jam")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Keluar Velora", "Tutup aplikasi sepenuhnya")

	c.mu.Lock()
	c.ready = true
	c.mStatus = mStatus
	c.mOpen = mOpen
	c.mStopJam = mStopJam
	c.mQuit = mQuit
	running := c.running
	url := c.url
	c.mu.Unlock()

	c.UpdateJamState(running, url)

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				if c.onOpen != nil {
					c.onOpen()
				}
			case <-mStopJam.ClickedCh:
				if c.onStopJam != nil {
					c.onStopJam()
				}
			case <-mQuit.ClickedCh:
				if c.onQuit != nil {
					c.onQuit()
				}
				return
			}
		}
	}()
}
