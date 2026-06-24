package dashboard

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"

	"github.com/123100123/lanlink/internal/agentserver"
	rscqr "rsc.io/qr"
)

func QRHandler(w http.ResponseWriter, r *http.Request) {
	s := agentserver.GetState()
	token := s.Token
	address := s.Address

	if token == "" || address == "" {
		http.Error(w, "no pairing data", http.StatusServiceUnavailable)
		return
	}

	payload := map[string]string{
		"t":  "l",
		"a":  address,
		"tk": token,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode payload", http.StatusInternalServerError)
		return
	}

	code, err := rscqr.Encode(string(data), rscqr.L)
	if err != nil {
		http.Error(w, "failed to generate QR", http.StatusInternalServerError)
		return
	}

	img := renderQRImage(code)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	png.Encode(w, img)
}

func renderQRImage(code *rscqr.Code) image.Image {
	size := code.Size
	scale := 8
	margin := 4
	total := (size + margin*2) * scale

	img := image.NewRGBA(image.Rect(0, 0, total, total))

	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}

	for y := 0; y < total; y++ {
		for x := 0; x < total; x++ {
			img.Set(x, y, white)
		}
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if code.Black(x, y) {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px := (x+margin)*scale + dx
						py := (y+margin)*scale + dy
						img.Set(px, py, black)
					}
				}
			}
		}
	}

	return img
}
