// Returns a span or series of spans made to format the price passed in
function formatPrice(price) {
	gold = Math.floor(price / 10000);
	silver = Math.floor((price % 10000) / 100);
	copper = price % 100;
	retVal = "";

	if (gold > 0) {
		retVal += "<span class='gw2Gold'>" + gold + "</span>";
	}

	if (silver > 0) {
		retVal += "<span class='gw2Silver'>" + silver + "</span>";
	}

	if (copper > 0 || retVal == "") {
		retVal += "<span class='gw2Copper'>" + copper + "</span>";
	}

	return retVal;
}