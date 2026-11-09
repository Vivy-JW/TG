using System;

namespace SpecDesignerCustomException
{
	// Token: 0x02000003 RID: 3
	public class FileFormatErrorException : Exception
	{
		// Token: 0x06000006 RID: 6 RVA: 0x0000208A File Offset: 0x0000028A
		public FileFormatErrorException()
		{
		}

		// Token: 0x06000007 RID: 7 RVA: 0x00002092 File Offset: 0x00000292
		public FileFormatErrorException(string message)
			: base(message)
		{
		}

		// Token: 0x06000008 RID: 8 RVA: 0x0000209B File Offset: 0x0000029B
		public FileFormatErrorException(string format, params object[] args)
			: base(string.Format(format, args))
		{
		}

		// Token: 0x06000009 RID: 9 RVA: 0x000020AA File Offset: 0x000002AA
		public FileFormatErrorException(string message, Exception innerException)
			: base(message, innerException)
		{
		}

		// Token: 0x0600000A RID: 10 RVA: 0x000020B4 File Offset: 0x000002B4
		public FileFormatErrorException(string format, Exception innerException, params object[] args)
			: base(string.Format(format, args), innerException)
		{
		}
	}
}
