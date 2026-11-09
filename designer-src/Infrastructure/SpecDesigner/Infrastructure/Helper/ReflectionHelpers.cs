using System;
using System.ComponentModel;
using System.Reflection;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x0200004B RID: 75
	public static class ReflectionHelpers
	{
		// Token: 0x06000246 RID: 582 RVA: 0x0000B190 File Offset: 0x00009390
		public static string GetCustomDescription(object objEnum)
		{
			FieldInfo field = objEnum.GetType().GetField(objEnum.ToString());
			DescriptionAttribute[] array = (DescriptionAttribute[])field.GetCustomAttributes(typeof(DescriptionAttribute), false);
			if (array.Length <= 0)
			{
				return objEnum.ToString();
			}
			return array[0].Description;
		}

		// Token: 0x06000247 RID: 583 RVA: 0x0000B1DB File Offset: 0x000093DB
		public static string Description(this Enum value)
		{
			return ReflectionHelpers.GetCustomDescription(value);
		}
	}
}
