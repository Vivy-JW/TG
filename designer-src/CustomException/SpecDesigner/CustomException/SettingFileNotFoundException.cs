using System;

namespace SpecDesigner.CustomException
{
	// Token: 0x02000002 RID: 2
	public class SettingFileNotFoundException : Exception
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public SettingFileNotFoundException()
		{
		}

		// Token: 0x06000002 RID: 2 RVA: 0x00002058 File Offset: 0x00000258
		public SettingFileNotFoundException(string message)
			: base(message)
		{
		}

		// Token: 0x06000003 RID: 3 RVA: 0x00002061 File Offset: 0x00000261
		public SettingFileNotFoundException(string format, params object[] args)
			: base(string.Format(format, args))
		{
		}

		// Token: 0x06000004 RID: 4 RVA: 0x00002070 File Offset: 0x00000270
		public SettingFileNotFoundException(string message, Exception innerException)
			: base(message, innerException)
		{
		}

		// Token: 0x06000005 RID: 5 RVA: 0x0000207A File Offset: 0x0000027A
		public SettingFileNotFoundException(string format, Exception innerException, params object[] args)
			: base(string.Format(format, args), innerException)
		{
		}
	}
}
