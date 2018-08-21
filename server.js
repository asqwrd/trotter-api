const fs = require("fs");
const app = require("express")();
const express = require("express");
const path = require("path");
const https = require("https");
const http = require("http");
const googleStorage = require("@google-cloud/storage");
const multer = require("multer");
const bodyParser = require("body-parser");
const format = require("util").format;
const request = require("request-promise");
const Vibrant = require("node-vibrant");
const admin = require("firebase-admin");

const serviceAccount = require("./serviceAccountKey.json");

admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});

var db = admin.firestore();
const country_codes = db.collection("countries_code");
const SHERPA_URL = "https://api.joinsherpa.com/v2/entry-requirements/",
  username = "VDLQLCbMmugvsOEtihQ9kfc6nQoeGd",
  password = "nIXaxALFPV0IiwNOvBEBrDCNSw3SCv67R4UEvD9r",
  SHERPA_AUTH =
    "Basic " + new Buffer(username + ":" + password).toString("base64");

app.use(function(req, res, next) {
  res.header("Access-Control-Allow-Origin", "*");
  res.header(
    "Access-Control-Allow-Methods",
    "GET,HEAD,OPTIONS,POST,PUT,DELETE"
  );
  res.header(
    "Access-Control-Allow-Headers",
    "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control"
  );
  next();
});

app.use(bodyParser.json());

const PROD_MODE = process.argv[2];
const API_KEY = "6SdxevLXN2aviv5g67sac2aySsawGYvJ6UcTmvWE";
let server = http.createServer(app);
if (PROD_MODE) {
  const hskey = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/privkey.pem");
  const hscert = fs.readFileSync(
    "/etc/letsencrypt/live/ajibade.me/fullchain.pem"
  );
  const hschain = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/chain.pem");
  const options = {
    key: hskey,
    cert: hscert,
    ca: hschain
  };
  server = https.createServer(options, app);
}

//Explore continent
app.get("/api/explore/continent/:continent_id", (req, res) => {
  const whatToSee = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?&level=poi&categories=sightseeing&parents=continent:${
      req.params.continent_id
    }&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const getPopularCities = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?rating=.0005:&level=city&parents=continent:${
      req.params.continent_id
    }&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const getAllCountries = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=country&parents=continent:${
      req.params.continent_id
    }&limit=60`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  Promise.all([whatToSee, getPopularCities, getAllCountries])
    .then(responses => {
      let points_of_interest = responses[0].data.places.reduce((acc, curr) => {
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

      let popular_cities = responses[1].data.places.reduce((acc, curr) => {
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

      let all_countries = responses[2].data.places.reduce((acc, curr) => {
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
      let popular_countries = all_countries.filter((country, index) => {
        return index < 10;
      });

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
});

//Explore country
app.get("/api/explore/countries/:country_id", (req, res) => {
  /*fs.readFile('./country_23.json', 'utf8', function (err, data) {
    if (err) throw err;
    res.send(JSON.parse(data));
  });*/
  const popular_destinations = request({
    method: "GET",
    uri: encodeURI(
      `https://api.sygictravelapi.com/1.1/en/places/list?&level=region|city|town|island|&parents=${
        req.params.country_id
      }&limit=20`
    ),
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const whatToSee = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/places/list?level=poi&categories=sightseeing&parents=${
      req.params.country_id
    }&limit=10`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const tours = request({
    method: "GET",
    uri: `https://api.sygictravelapi.com/1.1/en/tours/get-your-guide?sort_by=rating&parent_place_id=${
      req.params.country_id
    }&count=20`,
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  const country = request({
    method: "GET",
    uri: encodeURI(
      `https://api.sygictravelapi.com/1.1/en/places/${req.params.country_id}`
    ),
    json: true,
    headers: { "x-api-key": API_KEY }
  });

  Promise.all([country, popular_destinations, whatToSee, tours])
    .then(([parent, destinations, poi, tours]) => {
      const country_images = parent.data.place.main_media.media.reduce(
        (acc, curr) => {
          return [
            ...acc,
            {
              url: `${curr.url_template.replace(
                /\{[^\]]*?\}/g,
                "1200x800"
              )}.jpg`,
              title: curr.attribution.title,
              author: curr.attribution.author,
              other: curr.attribution.other
            }
          ];
        },
        []
      );

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
      };

      let points_of_interest = poi["data"].places.reduce((acc, curr) => {
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

      let popular_destinations = destinations["data"].places.reduce(
        (acc, curr) => {
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
        },
        []
      );

      let popular_tours = tours["data"].tours.reduce((acc, curr) => {
        return [
          ...acc,
          {
            sygic_id: curr.id,
            image: curr.photo_url.replace(/\[[^\]]*?\]/g, "21"),
            supplier: curr.supplier,
            url: curr.url,
            duration: curr.duration,
            rating: curr.rating,
            name: curr.title,
            name_suffix: curr.name_suffix,
            parent_ids: curr.parent_ids,
            description: curr.perex,
            price: curr.price
          }
        ];
      }, []);

      let country_color = Vibrant.from(country.image).getPalette();
      let country_name = country.name;
      if (country_name == "United States of America") {
        country_name = "United States";
      }
      let country_code = country_codes.doc(country_name).get();

      const data = new Promise((resolve, reject) => {
        resolve({
          country,
          popular_destinations,
          points_of_interest,
          popular_tours
        });
      });

      return Promise.all([data, country_color, country_code]);

      //res.send({popular_destinations, points_of_interest, popular_tours});
    })
    .then(([data, color, country_code]) => {
      let popular_destinations = data.popular_destinations;
      let points_of_interest = data.points_of_interest;
      let popular_tours = data.popular_tours;
      let country = data.country;
      country.color = `rgb(${color.Vibrant._rgb[0]},${color.Vibrant._rgb[1]},${
        color.Vibrant._rgb[2]
      })`;
      let visaData = country_code.data();

      if (visaData) {
        let visaCode = visaData.abbreviation;
        var visa = request({
          method: "GET",
          uri: `${SHERPA_URL}US-${visaCode}`,
          json: true,
          headers: { Authorization: SHERPA_AUTH }
        });
      }

      country.visa = null;

      let newProm = new Promise(resolve => {
        resolve({
          country,
          popular_destinations,
          points_of_interest,
          popular_tours
        });
      });

      if (!country_code.exists) {
        return newProm;
      }
      return Promise.all([newProm, visa]);

      //country.color = color;

      res.send({
        country,
        popular_destinations,
        points_of_interest,
        popular_tours
      });
    })
    .then(data => {
      if (data instanceof Array) {
        let {
          country,
          popular_destinations,
          points_of_interest,
          popular_tours
        } = data[0];
        country.visa = data[1];
        res.send({
          country,
          popular_destinations,
          points_of_interest,
          popular_tours
        });
      }

      res.send(data);
    })
    .catch(err => {
      console.log(err);
      res.send(err);
    });
});

//City
app.get("/api/explore/cities/:city_id", (req, res) => {});
app.get("/api/explore/cities/:city_id/sightseeing", (req, res) => {});

const uploadImageToStorage = file => {
  let prom = new Promise((resolve, reject) => {
    if (!file) {
      reject("No image file");
    }
    let newFileName = `${Date.now()}_${file.originalname}`;

    let fileUpload = bucket.file(newFileName);

    const blobStream = fileUpload.createWriteStream({
      metadata: {
        contentType: file.mimetype
      }
    });

    blobStream.on("error", error => {
      reject("Something is wrong! Unable to upload at the moment.");
    });

    blobStream.on("finish", e => {
      // The public URL can be used to directly access the file via HTTP.
      const url = format(
        `https://firebasestorage.googleapis.com/v0/b/${
          bucket.name
        }/o/${encodeURIComponent(fileUpload.name)}?alt=media`
      );
      resolve({ url, newFileName });
    });

    blobStream.end(file.buffer);
  });
  return prom;
};

const listen = server.listen(3002, function() {
  var host = server.address().address;
  var port = server.address().port;
  console.log("app listening at //%s:%s", host, port);
});

listen.timeout = 900000;
