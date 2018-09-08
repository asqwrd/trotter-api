import { Response, Request } from "express";
import * as request from "request-promise";

import { API_KEY } from "../server";
import { SygicPlace } from "./sygic-api.models";
import { BASE_SYGIC_API } from "./sygic-api.constants";

interface PlacesData {
  data: { places: SygicPlace[] };
}

interface Place {
  sygic_id: string;
  image: string;
  name: string;
  name_suffix: string;
  parent_ids: string[];
  description: string;
}

function sygicPlacesToInternal(sygicPlaces: SygicPlace[]): Place[] {
  return sygicPlaces.reduce((acc, curr) => {
    return [
      ...acc,
      {
        sygic_id: curr.id,
        image: curr.thumbnail_url,
        name: curr.name,
        name_suffix: curr.name_suffix,
        parent_ids: curr.parent_ids,
        description: curr.perex
      }
    ];
  }, []);
}

function constructPlacesRequest(continentID: string, queryParams: string) {
  return request.get({
    uri: `${BASE_SYGIC_API}/places/list?${queryParams}&parents=continent:${continentID}&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });
}

export const getContinent = (req: Request, res: Response) => {
  const continentID = req.params.continent_id;

  const whatToSee = constructPlacesRequest(continentID, "level=poi&categories=sightseeing");
  const getPopularCities = constructPlacesRequest(continentID, "rating=.0005:&level=city");
  // Why were we fetching 60 but dumping the last 50?
  const getAllCountries = constructPlacesRequest(continentID, "level=country");

  Promise.all([whatToSee, getPopularCities, getAllCountries])
    .then(responses => {
      const poiResponse = responses[0] as PlacesData;
      const citiesResponse = responses[1] as PlacesData;
      const countriesResponse = responses[2] as PlacesData;

      const points_of_interest = sygicPlacesToInternal(poiResponse.data.places);
      const popular_cities = sygicPlacesToInternal(citiesResponse.data.places);
      const all_countries = sygicPlacesToInternal(countriesResponse.data.places);

      // this is not really necessary if we set limit to 10 in the request above
      const popular_countries = all_countries.slice(0, 10);

      res.send({
        popular_countries,
        popular_cities,
        points_of_interest,
        all_countries
      });
    })
    .catch(err => {
      console.log(err);
      res.send(err);
    });
};
