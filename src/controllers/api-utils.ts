import { SygicPlace, Place } from "./sygic-api.models";
import { TriposoPlace, PlaceTriposo } from "./triposos-api-models";
import request from "request-promise";
import { BASE_SYGIC_API } from "./sygic-api.constants";
const API_KEY = "6SdxevLXN2aviv5g67sac2aySsawGYvJ6UcTmvWE";
const SHERPA_URL = "https://api.joinsherpa.com/v2/entry-requirements/",
  username = "VDLQLCbMmugvsOEtihQ9kfc6nQoeGd",
  password = "nIXaxALFPV0IiwNOvBEBrDCNSw3SCv67R4UEvD9r",
  TRIPOSO_ACCOUNT = '2ZWR5MHH',
  TRIPOSO_TOKEN = 'yan4ujbhzepr66ttsqxiqwcl38k3lx0w',
  SHERPA_AUTH = "Basic " + new Buffer(username + ":" + password).toString("base64");
 // /api/03/location.json?order_by=-trigram&count=1&fields=id,country_id&annotate=trigram:Port Harcourt&trigram=>=0.3


let countryCurrencies;

export function sygicPlacesToInternal(sygicPlaces: SygicPlace[]): Place[] {
  return sygicPlaces.reduce((acc, curr) => {
    return [
      ...acc,
      {
        sygic_id: curr.id,
        image: curr.thumbnail_url,
        name: curr.name,
        name_suffix: curr.name_suffix,
        parent_ids: curr.parent_ids,
        description_short: curr.perex,
        description: curr.description ? curr.description.text : null,
        level:curr.level,
        location:curr.location,
        address:curr.address,
        phone:curr.phone,
      }
    ];
  }, []);
}

export function triposoPlacesToInternal(triposoPlaces: TriposoPlace[]): PlaceTriposo[] {
  return triposoPlaces.reduce((acc, curr) => {
    return [
      ...acc,
      {
        triposo_id: curr.id,
        image: curr.images[0] ? curr.images[0].sizes.medium.url : null,
        name: curr.name,
        description_short: curr.snippet,
        description: curr.content && curr.content.sections ? curr.content.sections[0].body : null,
        level:'poi',
        location:{ lat: curr.coordinates.latitude, lng:curr.coordinates.longitude},
      }
    ];
  }, []);
}

export function triposoPlacesToLocations(triposoPlaces: TriposoPlace[]) {
  return triposoPlaces.reduce((acc, curr) => {
    return [
      ...acc,
      {
        lat:curr.coordinates.latitude,
        lng:curr.coordinates.longitude,
        title:curr.name,
        selected: false
      }
    ];
  }, []);
}

export function sygicPlacesToLocations(sygicPlaces: SygicPlace[]) {
  return sygicPlaces.reduce((acc, curr) => {
    return [
      ...acc,
      {
        lat:curr.location.lat,
        lng:curr.location.lng,
        title:curr.name,
        selected: false,
        bounding_box: curr.bounding_box
      }
    ];
  }, []);
}

export function constructPlacesRequest(id: string, queryParams: string, limit: number = 10) {
  return request.get({
    uri: `${BASE_SYGIC_API}/places/list?${queryParams}&parents=${id}&limit=${limit}`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });
}

export function getTriposoId(place:string) {
  return request.get({
    uri: `https://www.triposo.com/api/20180627/location.json?order_by=-trigram&count=1&fields=id,country_id&annotate=trigram:${place}&trigram=>=0.3&account=${TRIPOSO_ACCOUNT}&token=${TRIPOSO_TOKEN}`,
    json: true
  });
}

export function getTriposoPOI(id:string,tag_labels:string, count:number = 20) {
  return request.get({
    uri: encodeURI(`https://www.triposo.com/api/20180627/poi.json?location_id=${id}&tag_labels=${tag_labels}&count=${count}&fields=google_place_id,id,name,coordinates,tripadvisor_id,facebook_id,location_id,opening_hours,foursquare_id,snippet,content,images&account=${TRIPOSO_ACCOUNT}&token=${TRIPOSO_TOKEN}`),
    json: true
  });
}

export function getTriposoDestination(id:string) {
  return request.get({
    uri: `https://www.triposo.com/api/20180627/location.json?part_of=${id}&order_by=-score&fields=id,score,parent_id,country_id,structured_content,intro,name`,
    json: true
  });
}

export function constructPlaceRequest(id: string) {
  return request.get({
    uri: `${BASE_SYGIC_API}/places/${id}`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });
}

export function constructSherpaRequest(visaCode: string, citizenCode: string) {
  return request({
    method: "GET",
    uri: `${SHERPA_URL}${citizenCode}-${visaCode}`,
    json: true,
    headers: { Authorization: SHERPA_AUTH }
  });
}

export function constructSafetyRequest(visaCode: string) {
  return request({
    method: "GET",
    uri: `https://www.reisewarnung.net/api?country=${visaCode}`,
    json: true
  });
}

export function constructCurrencyConvertRequest(from: string, to: string) {
  return request({
    method: "GET",
    uri: `https://free.currencyconverterapi.com/api/v6/convert?q=${from}_${to}&compact=ultra`,
    json: true
  });
}

export async function getCountriesCurrenciesApi() {
  return request({
    method: "GET",
    uri: `https://free.currencyconverterapi.com/api/v6/countries`,
    json: true
  });
}

export function setCountriesCurrencies(currencies) {
  countryCurrencies = currencies;
}

export function getCountriesCurrencies() {
  return countryCurrencies;
}

export function removeDuplicates(myArr, prop) {
  return myArr.filter((obj, pos, arr) => {
    return arr.map(mapObj => mapObj[prop]).indexOf(obj[prop]) === pos;
  });
}
