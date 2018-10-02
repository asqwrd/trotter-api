import { Response, Request } from "express";

import {PlacesData} from "./sygic-api.models";
import {sygicPlacesToInternal, constructPlacesRequest} from "./api-utils";



export const getContinent = async (req: Request, res: Response) => {
  const continentID = req.params.continent_id;

  const whatToSee = constructPlacesRequest(continentID, "level=poi&categories=sightseeing");
  const getPopularCities = constructPlacesRequest(continentID, "rating=.0005:&level=city");

  const getAllCountries = constructPlacesRequest(continentID, "level=country", 60);

  const responses = await Promise.all([whatToSee, getPopularCities, getAllCountries])
  const poiResponse = responses[0] as PlacesData;
  const citiesResponse = responses[1] as PlacesData;
  const countriesResponse = responses[2] as PlacesData;

  const points_of_interest = sygicPlacesToInternal(poiResponse.data.places);
  const popular_cities = sygicPlacesToInternal(citiesResponse.data.places);
  const all_countries = sygicPlacesToInternal(countriesResponse.data.places);

  // displaying top 10 countries in the ui.  I did this to avoid having to make the same call twice.
  const popular_countries = all_countries.slice(0, 10);

  res.send({
    popular_countries,
    popular_cities,
    points_of_interest,
    all_countries
  });
};
