export interface SygicPlace {
  id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  name_suffix: string; // 'Havana, Cuba',
  url: string; //'https://go.sygic.com/travel/place?id=poi:24922',
  marker: string; // 'destination:borough',
  categories: string; // [ 'sightseeing', 'discovering' ],
  parent_ids: string[]; //[ 'region:69349', 'city:306', 'region:39455', 'region:2021320', 'country:51', 'continent:7' ],
  perex: string; // 'Old Havana is the city-center and one of the 15 municipalities forming Havana, Cuba.',
  thumbnail_url: string; // 'https://media-cdn.sygictraveldata.com/media/poi:24922' }
  color: string;
  level: string;
  address: string | null;
  admission: string | null;
  email: string | null;
  opening_hours: string | null;
  phone: string | null;
  area: number;
  location: {
    lat: number
    lng: number
  };
  bounding_box: {
    south: number
    west: number
    north: number
    east: number
  } | null
}

export interface PlacesData {
  data: { places: SygicPlace[] };
}

export interface Place {
  sygic_id: string;
  image: string;
  name: string;
  name_suffix: string;
  parent_ids: string[];
  description: string;
  color: string;
  visa: string;
  plugs: Object[];
  embassies: Object[];
  address: string | null;
  admission: string | null;
  email: string | null;
  opening_hours: string | null;
  phone: string | null;
  area: number;
  location: {
    lat: number
    lng: number
  };
  bounding_box: {
    south: number
    west: number
    north: number
    east: number
  } | null

}

export interface Country {
  sygic_id: string;
  image_usage: string;
  image: string;
  country_images;
  image_template: string;
  name: string;
  description: string;
  color: string;
  visa: string;
  plugs: Object[];
  embassies: Object[];
  emergency_numbers: { ambulance: String[], police: String[], fire: String[], dispatch: String[] };
  currency: { converted_currency: number, converted_unit: string, unit: string };

}
