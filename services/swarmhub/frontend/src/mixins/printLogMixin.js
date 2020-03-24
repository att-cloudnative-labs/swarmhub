import moment from 'moment';
import stripAnsi from 'strip-ansi'

export const printLogMixin = {
  methods: {
    logPrint(log) {
      var logprint = "";
      if (log.Output != "") {
        logprint =
          moment(log.Timestamp).format("MMM D, YYYY h:mm:ssA") +
          ": " +
          stripAnsi(log.Output);
      }
      return logprint;
    },
  }
}