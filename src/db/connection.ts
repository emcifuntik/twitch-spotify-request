import { createConnection } from 'mysql2';

const connection = createConnection({
  host: process.env.SQL_HOST,
  database: process.env.SQL_DATABASE,
  user: process.env.SQL_USER,
  password: process.env.SQL_PASSWORD
});

export default connection;