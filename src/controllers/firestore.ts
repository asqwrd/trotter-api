import admin from "firebase-admin";
import NodeGeocoder from "node-geocoder";

const options = {
  provider: "google",

  // Optional depending on the providers
  httpAdapter: "https", // Default
  apiKey: "AIzaSyBEb1lr2C8pLBcP20y2j77h4C89RuQE1v8", // for Mapquest, OpenCage, Google Premier
  formatter: null // 'gpx', 'string', ...
};
const settings = { /* your settings... */ timestampsInSnapshots: true };
export const geocoder = NodeGeocoder(options);


const serviceAccount = require("../../serviceAccountKey.json");
admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});

const firestore = admin.firestore();
firestore.settings(settings);

export const db = firestore;

