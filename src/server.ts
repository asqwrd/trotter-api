import { Request, Response, NextFunction } from "express";

import { getContinent } from "./controllers/continent";
import { getCountry } from "./controllers/country";
import { getCity } from "./controllers/city";

import fs from "fs";
import express from "express";
import path = require("path");
import https from "https";
import http from "http";
import googleStorage from "@google-cloud/storage";
import multer from "multer";
import bodyParser = require("body-parser");

const app = express();

app.use(function(req: Request, res: Response, next: NextFunction) {
  res.header("Access-Control-Allow-Origin", "*");
  res.header("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT,DELETE");
  res.header(
    "Access-Control-Allow-Headers",
    "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control"
  );
  next();
});

app.use(bodyParser.json());

const PROD_MODE = process.argv[2];
let server;
if (PROD_MODE) {
  const hskey = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/privkey.pem");
  const hscert = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/fullchain.pem");
  const hschain = fs.readFileSync("/etc/letsencrypt/live/ajibade.me/chain.pem");
  const options = {
    key: hskey,
    cert: hscert,
    ca: hschain
  };
  server = https.createServer(options, app);
} else {
  server = http.createServer(app);
}

//Explore continent
app.get("/api/explore/continent/:continent_id", getContinent);

//Explore country
app.get("/api/explore/countries/:country_id", getCountry)
    
//City
app.get("/api/explore/cities/:city_id", getCity);

//This could is snippets for uploading image. I will eventually add profile images once we flesh out user management
// const uploadImageToStorage = file => {
//   let prom = new Promise((resolve, reject) => {
//     if (!file) {
//       reject("No image file");
//     }
//     let newFileName = `${Date.now()}_${file.originalname}`;

//     let fileUpload = bucket.file(newFileName);

//     const blobStream = fileUpload.createWriteStream({
//       metadata: {
//         contentType: file.mimetype
//       }
//     });

//     blobStream.on("error", error => {
//       reject("Something is wrong! Unable to upload at the moment.");
//     });

//     blobStream.on("finish", e => {
//       // The public URL can be used to directly access the file via HTTP.
//       const url = format(
//         `https://firebasestorage.googleapis.com/v0/b/${bucket.name}/o/${encodeURIComponent(
//           fileUpload.name
//         )}?alt=media`
//       );
//       resolve({ url, newFileName });
//     });

//     blobStream.end(file.buffer);
//   });
//   return prom;
// };

const listen = server.listen(3002, function() {
  var host = server.address().address;
  var port = server.address().port;
  console.log("app listening at //%s:%s", host, port);
});

listen.timeout = 900000;
