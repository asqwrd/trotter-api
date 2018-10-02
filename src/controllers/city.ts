import { Response, Request } from "express";

import {PlacesData} from "./sygic-api.models";
import {sygicPlacesToInternal, constructPlacesRequest, constructPlaceRequest, sygicPlacesToLocations,   getTriposoId,
  getTriposoPOI,
  getTriposoDestination,triposoPlacesToInternal,triposoPlacesToLocations} from "./api-utils";


async function getCityTriposo(location_id){
  const sightseeing_request_triposo = getTriposoPOI(location_id,'sightseeing|sight|topattractions');
  const discovering_request_triposo = getTriposoPOI(location_id,'museums|tours|walkingtours|transport|private_tours|celebrations|hoponhopoff|air|architecture|multiday|touristinfo|forts');
  const playing_request_triposo= getTriposoPOI(location_id,'amusementparks|golf|iceskating|kayaking|sporttickets|sports|surfing|cinema|zoos');
  const relaxing_request_triposo = getTriposoPOI(location_id,'beaches|camping|wildlife|fishing|relaxinapark');
  const eat_request_triposo = getTriposoPOI(location_id,'eatingout|breakfast|coffeeandcake|lunch|dinner');
  const shop_request_triposo = getTriposoPOI(location_id,'do|shopping');
  const nightlife_request_triposo = getTriposoPOI(location_id,'nightlife|comedy|drinks|dancing|pubcrawl|redlight|musicandshows|celebrations|foodexperiences|breweries|showstheatresandmusic');

  const pois = await Promise.all([
    sightseeing_request_triposo,
    discovering_request_triposo,
    playing_request_triposo,
    relaxing_request_triposo,
    eat_request_triposo,
    shop_request_triposo,
    nightlife_request_triposo,
  ]);
  const [see, discover, playing, relax,eat,shop,nightlife] = pois;
  return {
    see : see.results,
    discover: discover.results,
    playing: playing.results, 
    relax: relax.results,
    shop: shop.results,
    nightlife: nightlife.results,
    eat:eat.results

  }
}

export const getCity = async (req: Request, res: Response) => {
  const id = req.params.city_id;


  /*const see_request = constructPlacesRequest(id, "level=poi&categories=sightseeing", 20);

  const discovering_request = constructPlacesRequest(id, "level=poi&categories=discovering", 20);

  const playing_request = constructPlacesRequest(id, "level=poi&categories=playing", 20);

  const eating_request = constructPlacesRequest(id, "level=poi&categories=eating", 20);
  const shopping_request = constructPlacesRequest(id, "level=poi&categories=shopping", 20);*/
  const city_request = await constructPlaceRequest(id);


  //const responses = await Promise.all([see_request, discovering_request, playing_request, eating_request, shopping_request, city_request])
  /*const seeing = responses[0] as PlacesData;
  const discovering = responses[1] as PlacesData;
  const playing = responses[2] as PlacesData;
  const eating = responses[3] as PlacesData;
  const shopping = responses[4] as PlacesData;*/
  const cityData = city_request;

  /*const see = sygicPlacesToInternal(seeing.data.places);
  const discover = sygicPlacesToInternal(discovering.data.places);
  const play = sygicPlacesToInternal(playing.data.places);
  const eat = sygicPlacesToInternal(eating.data.places);
  const shop = sygicPlacesToInternal(shopping.data.places);

  const see_locations = sygicPlacesToLocations(seeing.data.places);
  const discover_locations = sygicPlacesToLocations(discovering.data.places);
  const play_locations = sygicPlacesToLocations(playing.data.places);
  const eat_locations = sygicPlacesToLocations(eating.data.places);
  const shop_locations = sygicPlacesToLocations(shopping.data.places);*/

  const city = {
    sygic_id: cityData.data.place.id,
    image_usage: cityData.data.place.main_media.usage,
    image: cityData.data.place.main_media.media[0].url,
    image_template: `${cityData.data.place.main_media.media[0].url_template.replace(
      /\{[^\]]*?\}/g,
      "1200x800"
    )}.jpg`,
    name: cityData.data.place.name,
    description: cityData.data.place.perex,
    location: cityData.data.place.location,
    bounding_box: cityData.data.place.bounding_box,
  } as any;

  const triposoData = await getCityTriposo(city.name);
  let see = triposoPlacesToInternal(triposoData.see);
  let eat = triposoPlacesToInternal(triposoData.eat);
  let relax = triposoPlacesToInternal(triposoData.relax);
  let shop = triposoPlacesToInternal(triposoData.shop);
  let play = triposoPlacesToInternal(triposoData.playing);
  let nightlife = triposoPlacesToInternal(triposoData.nightlife);
  let discover = triposoPlacesToInternal(triposoData.discover);

  const see_locations = triposoPlacesToLocations(triposoData.see);
  const discover_locations = triposoPlacesToLocations(triposoData.discover);
  const play_locations = triposoPlacesToLocations(triposoData.playing);
  const eat_locations = triposoPlacesToLocations(triposoData.eat);
  const shop_locations = triposoPlacesToLocations(triposoData.shop);
  const nightlife_locations = triposoPlacesToLocations(triposoData.nightlife);
  const relax_locations = triposoPlacesToLocations(triposoData.relax);

  




  // displaying top 10 countries in the ui.  I did this to avoid having to make the same call twice.

  res.send({
    city,
    see,
    eat,
    relax,
    shop,
    play,
    nightlife,
    discover,
    see_locations,
    discover_locations,
    play_locations,
    eat_locations,
    shop_locations,
    nightlife_locations,
    relax_locations
  });
};
