import { Response, Request } from "express";

import {PlacesData} from "./sygic-api.models";
import {sygicPlacesToInternal, constructPlacesRequest, getTriposoId, getTriposoDestination, triposoPlacesToInternal} from "./api-utils";
import { TriposoPlace } from "./triposos-api-models";
import _ from 'lodash';



export const getContinent = async (req: Request, res: Response) => {
  const continentID = req.params.continent_id;


  const getAllCountries =  await constructPlacesRequest(continentID, "level=country", 60);

  const countriesResponse = getAllCountries as PlacesData;
  

  let all_countries = sygicPlacesToInternal(countriesResponse.data.places);
  // displaying top 10 countries in the ui.  I did this to avoid having to make the same call twice.
  const popular_countries = all_countries.slice(0, 5);
  const cities = popular_countries.reduce((acc, curr)=>{
    return [...acc, getTriposoId(curr.name)]
  },[])

  const cities_response = await Promise.all(cities);
  const cities_dest = cities_response.reduce((acc,curr) => {

    return [...acc, getTriposoDestination(curr.results[0].id,2) ]
  },[]);

  const popular_cities_response =  await Promise.all(cities_dest);
  let popular_cities = popular_cities_response.reduce((acc:Object[],curr) =>{ 
    const results = triposoPlacesToInternal(curr['results']);
    return [...acc, ...results]
  },[]);

  popular_cities = _.orderBy(popular_cities,['score'],['desc']);
  all_countries = _.sortBy(all_countries,'name');

  res.send({
    popular_cities,
    all_countries
  });
};
