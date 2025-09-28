export const appInfo = {
  // learn more about this on https://supertokens.com/docs/thirdpartyemailpassword/appinfo
  appName: 'Nixopus',
  apiDomain: process.env.API_URL?.replace('/api', '') || 'http://localhost:8080',
  // TODO: Check if this works properly in production, else keep an environment variable for the website domain
  websiteDomain: window?.location?.hostname + ':' + process.env.NEXT_PUBLIC_PORT || 'http://localhost:3000',
  apiBasePath: '/auth',
  websiteBasePath: '/auth'
};
