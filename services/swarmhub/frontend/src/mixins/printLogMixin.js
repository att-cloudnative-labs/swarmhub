import moment from 'moment';

export const printLogMixin = {
  methods: {
    logPrint(log) {
      var logprint = "";
      if (log.Output != "") {
        logprint =
          moment(log.Timestamp).format("MMM D, YYYY h:mm:ssA") +
          ": " +
          log.Output;
      }
      return logprint;
    },
  }
}