module.exports = {
  quiz: {
    input: '../swagger.yaml',
    output: {
      target: './src/api/api.ts',
      client: 'react-query',
      mock: true,
      prettier: true,
      override: {
        mutator: {
          path: './src/api/axios.ts',
          name: 'customInstance',
        },
        query: {
          useQuery: true,
          signal: true,
        },
      },
    },
    hooks: {
      afterAllFilesWrite: 'prettier --write',
    },
  },
};
