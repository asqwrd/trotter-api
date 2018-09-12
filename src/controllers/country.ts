import { Response, Request } from "express";
import Vibrant from "node-vibrant";

import { Country } from "./sygic-api.models";
import { db, geocoder } from "./firestore";
import {
  sygicPlacesToInternal,
  constructPlacesRequest,
  removeDuplicates,
  constructPlaceRequest,
  constructSherpaRequest,
  constructSafetyRequest,
  getCountriesCurrencies,
  getCountriesCurrenciesApi,
  setCountriesCurrencies,
  constructCurrencyConvertRequest
} from "./api-utils";
import util from "util";

const format = util.format;
const citizenCode = "US";
const citizenCountry = "United States";
const country_codes = db.collection("countries_code");
const emergency_numbers_db = db.collection("emergency_numbers");
const plugs_db = db.collection("plugs");

(async function(){
  const currencies = await getCountriesCurrenciesApi();
  setCountriesCurrencies(currencies.results);
})()



const passport_blankpages_map = {
  NOT_REQUIRED: "You do not need to have any blank pages in your passport.",
  ONE: "You need at least one blank page in your passport.",
  ONE_PER_ENTRY: "You need one blank page per entry.",
  SPACE_FOR_STAMP: "You need space for your passport to be stamped.",
  TWO: "You need two blank pages in your passport.",
  TWO_CONSECUTIVE_PER_ENTRY: "You need two consecutive blank pages in your passport",
  TWO_PER_ENTRY: "You need two blank pages per entry"
};

const passport_validity_map = {
  DURATION_OF_STAY: "Your passport must be valid for the duration of your stay in this country.",
  ONE_MONTH_AFTER_ENTRY: "Your passport must be valid for one month after entering this counrty.",
  SIX_MONTHS_AFTER_DURATION_OF_STAY:
    "Your passport must be valid on entry and for six months after the duration of your stay in this country.",
  SIX_MONTHS_AFTER_ENTRY:
    "Your passport must be valid on entry and six months after the date of enrty.",
  SIX_MONTHS_AT_ENTRY:
    "Your passport must be valid for at least six months before entering this country.",
  THREE_MONTHS_AFTER_DURATION_OF_STAY:
    "Your passport must be valid on entry and for three months after the duration of your stay in this country",
  THREE_MONTHS_AFTER_ENTRY:
    "Your passport must be valid on entry and for three months after entering this country",
  VALID_AT_ENTRY: "Your passport must be valid on entry",
  THREE_MONTHS_AFTER_DEPARTURE:
    "Your passport must be valid on entry and three months after your departure date.",
  SIX_MONTHS_AFTER_DEPARTURE:
    "Your passport must be valid on entry and six months after your departure date."
};

const countries_with_states = {
  Argentina: true,
  Australia: true,
  Austria: true,
  Belgium: true,
  "Bosnia and Herzegovina": true,
  Brazil: true,
  Canada: true,
  Comoros: true,
  Ethiopia: true,
  Germany: true,
  India: true,
  Iraq: true,
  Malaysia: true,
  Mexico: true,
  Micronesia: true,
  Nepal: true,
  Nigeria: true,
  Pakistan: true,
  Russia: true,
  "Saint Kitts": true,
  Somalia: true,
  "South Sudan": true,
  Sudan: true,
  Switzerland: true,
  "United Arab Emirates": true,
  "United States of America": true,
  Venezuela: true
};

async function getCountryResearch(req) {
  const id = req.params.country_id;

  const popular_destinations_request = constructPlacesRequest(
    id,
    "level=region|city|town|island",
    20
  );

  const sightseeing_request = constructPlacesRequest(
    id,
    "level=poi&categories=sightseeing",
    20
  );

  const discovering_request = constructPlacesRequest(
    id,
    "level=poi&categories=discovering",
    20
  );

  const playing_request = constructPlacesRequest(id, "level=poi&categories=playing", 20);

  const relaxing_request = constructPlacesRequest(
    id,
    "level=poi&categories=relaxing",
    20
  );

  const country_request = constructPlaceRequest(id);

  return Promise.all([
    country_request,
    popular_destinations_request,
    sightseeing_request,
    discovering_request,
    playing_request,
    relaxing_request
  ]).catch((error)=>{console.log(error)});
}

function formatCountryResearch(responses) {
  // const [parent, destinations, sights, discover, play, relax] = responses;
  const parent = responses[0] as any;
  const destinations = responses[1] as any;
  const sights = responses[2] as any;
  const discover = responses[3] as any;
  const play = responses[4] as any;
  const relax = responses[5] as any;

  const country_images = parent.data.place.main_media.media.reduce((acc, curr) => {
    return [
      ...acc,
      {
        url: `${curr.url_template.replace(/\{[^\]]*?\}/g, "1200x800")}.jpg`,
        title: curr.attribution.title,
        author: curr.attribution.author,
        other: curr.attribution.other
      }
    ];
  }, []);

  let country = {
    sygic_id: parent.data.place.id,
    image_usage: parent.data.place.main_media.usage,
    image: parent.data.place.main_media.media[0].url,
    country_images,
    image_template: `${parent.data.place.main_media.media[0].url_template.replace(
      /\{[^\]]*?\}/g,
      "1200x800"
    )}.jpg`,
    name: parent.data.place.name,
    description: parent.data.place.perex
  } as Country;

  let sightseeing = sygicPlacesToInternal(sights["data"].places);

  let discovering = sygicPlacesToInternal(discover["data"].places);

  let playing = sygicPlacesToInternal(play["data"].places);

  let relaxing = sygicPlacesToInternal(relax["data"].places);

  let popular_destinations = sygicPlacesToInternal(destinations["data"].places);

  return {
    country,
    popular_destinations,
    sightseeing,
    discovering,
    playing,
    relaxing
  };
}

async function countryUICalls(data, req) {
  const id = req.params.country_id;
  let country_name = data.country.name;
  if (country_name == "United States of America") {
    country_name = "United States";
  }
  let country_code = country_codes.doc(country_name).get();
  let country_color = Vibrant.from(data.country.image).getPalette();
  let plugs = plugs_db.where("country", "==", country_name).get();

  if (countries_with_states[data.country.name]) {
    const states = constructPlacesRequest(id, "level=state", 100);

    return Promise.all([country_color, country_code, plugs, states]).catch((error)=>{console.log(error)}) as Promise<any[]>;
  }

  return Promise.all([country_color, country_code, plugs]).catch((error)=>{console.log(error)}) as Promise<any[]>;
}

async function visaInfo(visaData, visaCode, country_code, currency) {
  let visa, safety, emergency_numbers,currency_convert;

  if (visaData && visaCode !== citizenCode) {
    visa = constructSherpaRequest(visaCode, citizenCode);
  }

  if (visaCode) {
    safety = constructSafetyRequest(visaCode);
    emergency_numbers = emergency_numbers_db.doc(visaCode).get();
    currency_convert = constructCurrencyConvertRequest(currency.from.currencyId,currency.to.currencyId);
  }

  if (!country_code.exists || visaCode == citizenCode) {
    return Promise.all([safety, emergency_numbers,currency_convert]).catch((error)=>{console.log(error)}) as Promise<any>;
  }

  return Promise.all([safety, emergency_numbers, currency_convert, visa]).catch((error)=>{console.log(error)});
}

function formatVisa(visaData) {
  let visa = visaData;
  visa.visa = visa.visa ? visa.visa[0] : null;
  visa.passport = visa.passport ? visa.passport : { passport_validity: null, blank_pages: null };
  let passport_valid = visa.passport.passport_validity ? visa.passport.passport_validity : null;
  let blank_pages = visa.passport.blank_pages ? visa.passport.blank_pages : null;
  visa.passport.passport_validity =
    passport_valid && passport_validity_map[passport_valid]
      ? passport_validity_map[passport_valid]
      : `${
          passport_validity_map["VALID_AT_ENTRY"]
        } Make sure to check for additional requirements.`;

  visa.passport.blank_pages =
    blank_pages && passport_blankpages_map[blank_pages]
      ? passport_blankpages_map[blank_pages]
      : "To be safe make sure to have at least one blank page in your passport.";
  return visa;
}

function formatSafety(rating) {
  let advice = "No safety information is available for this country.";
  if (rating >= 0 && rating < 1) {
    advice = "Travelling in this country is relatively safe.";
  } else if (rating >= 1 && rating < 2.5) {
    advice =
      "Travelling in this country is relatively safe. Higher attention is advised when traveling here due to some areas being unsafe.";
  } else if (rating >= 2.5 && rating < 3.5) {
    advice =
      "This country can be unsafe.  Warnings often relate to specific regions within this country. However, high attention is still advised when moving around. Trotter also recommends traveling to this country with someone who is familiar with the culture and area.";
  } else if (rating >= 3.5 && rating < 4.5) {
    advice =
      "Travel to this country should be reduced to a necessary minimum and be conducted with good preparation and high attention. If you are not familiar with the area it is recommended you travel with someone who knows the area well.";
  } else if (rating >= 4.5) {
    advice =
      "It is unsafe to travel to this country.  Trotter advises against traveling here.  You risk high chance of danger to you health and life.";
  }
  return advice;
}

function addEmergencyNumber(emergency_numbers) {
  let { ambulance, police, dispatch, fire } = emergency_numbers;

  ambulance = ambulance.all.filter(item => {
    return item != null && item != undefined && item != "";
  });
  police = police.all.filter(item => {
    return item != null && item != undefined && item != "";
  });
  dispatch = dispatch.all.filter(item => {
    return item != null && item != undefined && item != "";
  });
  fire = fire.all.filter(item => {
    return item != null && item != undefined && item != "";
  });

  return {
    ambulance,
    police,
    dispatch,
    fire
  };
}

async function getEmbassy(req) {
  const id = req.params.country_id;
  const embassy = constructPlacesRequest(id, `tags=Embassy&query=${citizenCountry}&level=poi`, 20) ;
  return embassy;
}

function formatEmbassy(embassy, embassy_names) {
  let embassies = embassy.reduce((acc, curr, index) => {
    return [
      ...acc,
      {
        address: curr[0].formattedAddress,
        lat: curr[0].latitude,
        lng: curr[0].longitude,
        name: embassy_names[index]
      }
    ];
  }, []);

  return removeDuplicates(embassies, "address");
}

async function getAddresses(embassies) {
  let addresses;
  if (embassies && embassies.data.places.length > 0) {
    addresses = embassies.data.places.reduce((acc, curr) => {
      return [...acc, geocoder.reverse({ lat: curr.location.lat, lon: curr.location.lng })];
    }, []);
    const embassy_names = embassies.data.places.reduce((acc, curr) => {
      return [...acc, curr.name];
    }, []);
    return Promise.all([embassy_names, ...addresses]).catch((error)=>{console.log(error)});
  }
  return;
}

export const getCountry = async (req: Request, res: Response) => {
  const currencies = getCountriesCurrencies();
  const citizenCurrency = currencies[citizenCode];
  let responses = await getCountryResearch(req);
  let data = formatCountryResearch(responses);
  let extraCountry = await countryUICalls(data, req);
  let [color, country_code, plugsRes, ...args]: any[] = extraCountry;

  let states = args[0];
  let popular_destinations = data.popular_destinations;
  let sightseeing = data.sightseeing;
  let discover = data.discovering;
  let play = data.playing;
  let relax = data.relaxing;
  let country = data.country;
  country.color = `rgb(${color.Vibrant._rgb[0]},${color.Vibrant._rgb[1]},${color.Vibrant._rgb[2]})`;
  let plugs = [];
  plugsRes.forEach(plug => {
    plugs = [...plugs, plug.data()];
  }, []);
  country.plugs = plugs;
  let visaData = country_code.data();
  let visaCode = visaData.abbreviation;
  
  country.visa = null;

  if (states) {
    states = sygicPlacesToInternal(states["data"].places);
  }
  let currencyObj = {
    from: citizenCurrency,
    to: currencies[visaCode]
  }
  let helpfulInfo = await visaInfo(visaData, visaCode, country_code, currencyObj);

  let [safetyData, emergency_numbersData, converted_currency, visaStuff] = helpfulInfo;

  country.currency = {
    converted_currency: converted_currency[`${currencyObj.from.currencyId}_${currencyObj.to.currencyId}`],
    converted_unit: currencyObj.to,
    unit: citizenCurrency
  }

  let visa = visaStuff;
  if (visa) {
    country.visa = formatVisa(visa);
  }

  let { rating } = safetyData.data.situation;
  let advice = formatSafety(rating);

  let safety = {
    rating,
    advice
  };
  let emergency_numbers = emergency_numbersData.data();

  country.emergency_numbers = addEmergencyNumber(emergency_numbers);

  let embassiesRes = await getEmbassy(req);
  let embassies = await getAddresses(embassiesRes);
  if (embassies) {
    let embassy_names = embassies.splice(0, 1)[0];
    data.country.embassies = formatEmbassy(embassies, embassy_names);
  }

  res.send({
    country,
    popular_destinations,
    sightseeing,
    discover,
    play,
    relax,
    states,
    safety
  });
};
