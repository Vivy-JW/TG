using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000040 RID: 64
	public class CodeSampleSourceFactory
	{
		// Token: 0x06000178 RID: 376 RVA: 0x00007350 File Offset: 0x00005550
		public static List<CodeSamepleModel> Parse(string source)
		{
			List<CodeSamepleModel> list = new List<CodeSamepleModel>();
			IEnumerable<XElement> enumerable = from query in XDocument.Parse(source).Descendants("sample")
				select (query);
			foreach (XElement xelement in enumerable)
			{
				CodeSamepleModel codeSamepleModel = new CodeSamepleModel();
				codeSamepleModel.Content = xelement.Value;
				foreach (XAttribute xattribute in xelement.Attributes())
				{
					string text;
					if ((text = xattribute.Name.LocalName.ToLower()) != null)
					{
						if (!(text == "id"))
						{
							if (!(text == "desc"))
							{
								string text2;
								if (!(text == "range"))
								{
									if (text == "show")
									{
										if (xattribute.Value.Equals("Y", StringComparison.InvariantCultureIgnoreCase))
										{
											codeSamepleModel.IsShow = true;
										}
										else
										{
											codeSamepleModel.IsShow = false;
										}
									}
								}
								else if ((text2 = xattribute.Value.ToLower()) != null)
								{
									if (!(text2 == "sub"))
									{
										if (!(text2 == "lib"))
										{
											if (text2 == "erp")
											{
												codeSamepleModel.Range = RangeEnum.ERP;
											}
										}
										else
										{
											codeSamepleModel.Range = RangeEnum.LIB;
										}
									}
									else
									{
										codeSamepleModel.Range = RangeEnum.SUB;
									}
								}
							}
							else
							{
								codeSamepleModel.Description = xattribute.Value;
								codeSamepleModel.Description = codeSamepleModel.Description.Replace("\n", Environment.NewLine);
							}
						}
						else
						{
							codeSamepleModel.ID = xattribute.Value;
						}
					}
				}
				list.Add(codeSamepleModel);
			}
			return list;
		}
	}
}
