using System;
using System.Collections.Generic;

namespace Microsoft.Shell
{
	// Token: 0x02000024 RID: 36
	public interface ISingleInstanceApp
	{
		// Token: 0x06000246 RID: 582
		bool SignalExternalCommandLineArgs(IList<string> args);
	}
}
