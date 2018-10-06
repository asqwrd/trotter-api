export interface TriposoPlace {
  id: string; // 'poi:24922'
  location_id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  opening_hours: string | null;
  intro: string;
  coordinates: {
    latitude: number
    longitude: number
  };
  description: string;
  content:{
    sections: {
      body:string,
    }[]
  };
  images:{
    owner_url: string,
    sizes: {
      medium:{
        url:string;
      }
    }
  }[];
  facebook_id: string;
  foursquare_id: string;
  google_place_id: string;
  tripadvisor_id: string;
  snippet:string;
  score:number;
  best_for:string;
  booking_info:string;
  price_tier:string;
}

export interface PlacesData {
  data: { places: TriposoPlace[] };
}

export interface PlaceTriposo {
  id: string; // 'poi:24922'
  location_id: string; // 'poi:24922'
  name: string; // 'Old Havana',
  opening_hours: string | null;
  intro: string;
  location: {
    lat: number
    lng: number
  };
  description: string;
  content:{
    sections: {
      body:string,
    }[]
  };
  images:{
    owner_url: string,
    sizes: {
      medium:{
        url:string;
      }
    }
  }[];
  facebook_id: string;
  foursquare_id: string;
  google_place_id: string;
  snippet:string;
  score:number;
  tripadvisor_id: string;
  price_tier: string;
  booking_info: string;
  best_for: string;

}

