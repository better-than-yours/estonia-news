package tests

import (
	"estonia-news/service"

	"github.com/stretchr/testify/assert"
)

func (t *SuiteTest) Test_Message_FormatText() {
	message1 := `Ööl vastu 24.window.addEventListener("message", function msg(event) {if (event.data != "" && event.data.indexOf("px") > -1) {var height = event.data.split(":"); var iframe = document.getElementById(height[0]);if(iframe){iframe.style.height = height[1]; iframe.style.width = "100%"; iframe.style.border = 0; iframe.style.overflow = "hidden"; iframe.parentElement.parentElement.style.maxWidth = "100%";}}}, false);
	#bookmark { border: 0; overflow: hidden;} .device-desktop .superarticle #bookmark { padding-left: 30px !important; max-width: 750px !important;}

	`
	assert.Equal(t.T(), "Ööl vastu 24.", service.CleanUpText(message1))

	message2 := `


 


 

 



 




	View this post on Instagram


	 



	 

	 

	 



	 

	 



	 

	 

	 




	 

	 


	A post shared by Pirinen & Salo (@pirinensalo)

	Soome arhitektuuribüroo Pirinen & Salo disainis Porovesi järve kaldale väga erilise hüti.`
	assert.Equal(t.T(), "View this post on Instagram\nA post shared by Pirinen & Salo (@pirinensalo)\nSoome arhitektuuribüroo Pirinen & Salo disainis Porovesi järve kaldale väga erilise hüti.", service.CleanUpText(message2))

	message3 := `window.addEventListener("message", function msg(event) {if (event.data != "" && event.data.indexOf("px") > -1) {var height = event.data.split(":"); var iframe = document.getElementById(height[0]);if(iframe){iframe.style.height = height[1]; iframe.style.width = "100%"; iframe.style.border = 0; iframe.style.overflow = "hidden"; iframe.parentElement.parentElement.style.maxWidth = "100%";}}}, false);
	#bookmark { border: 0; overflow: hidden;} .device-desktop .superarticle #bookmark { padding-left: 30px !important; max-width: 750px !important;}

	Ööl vastu 24.`
	assert.Equal(t.T(), "Ööl vastu 24.", service.CleanUpText(message3))

	message4 := `Ööl vastu 24.<img src="http://feeds.feedburner.com/~r/delfiuudised/~4/t4DO-Uy3On4" height="1" width="1" alt=""/>`
	assert.Equal(t.T(), "Ööl vastu 24.", service.CleanUpText(message4))
}
