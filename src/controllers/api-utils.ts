import { SygicPlace, Place } from "./sygic-api.models";
import request from "request-promise";
import { BASE_SYGIC_API } from "./sygic-api.constants";
const API_KEY = "6SdxevLXN2aviv5g67sac2aySsawGYvJ6UcTmvWE";
const SHERPA_URL = "https://api.joinsherpa.com/v2/entry-requirements/",
  username = "VDLQLCbMmugvsOEtihQ9kfc6nQoeGd",
  password = "nIXaxALFPV0IiwNOvBEBrDCNSw3SCv67R4UEvD9r",
  SHERPA_AUTH = "Basic " + new Buffer(username + ":" + password).toString("base64");

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
        description: curr.perex,
        level:curr.level,
        location:curr.location,
        address:curr.address,
        phone:curr.phone,
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
